package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"ppeelink/models"
	"ppeelink/node"
	"ppeelink/utils"

	"gorm.io/gorm"
)

// nodesFromSubscriptionBody parses a subscription body in any of the formats
// airports commonly return: Clash/Mihomo YAML, a base64 encoded link list, or
// a plain newline/comma separated link list. It returns the parsed nodes and
// the detected format label for diagnostics.
func nodesFromSubscriptionBody(body []byte) ([]models.Node, string, error) {
	if node.IsClashConfig(body) {
		clashNodes, err := node.ParseClashToNodes(body)
		if err != nil {
			return nil, "clash-yaml", err
		}
		nodesList := make([]models.Node, 0, len(clashNodes))
		for _, cn := range clashNodes {
			nodesList = append(nodesList, models.Node{Name: cn.Name, Link: cn.Link})
		}
		return nodesList, "clash-yaml", nil
	}

	text := string(body)
	format := "link-list"
	if decoded := node.Base64Decode(text); decoded != "" && decoded != text {
		text = decoded
		format = "base64-link-list"
	}

	lines := strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' })
	valid := make([]models.Node, 0, len(lines))
	skipped := 0
	for _, link := range lines {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		if !strings.Contains(link, "://") {
			skipped++
			continue
		}
		n := models.Node{Link: link}
		parsed, err := DocodeNodeName(&n)
		if err != nil || parsed.Name == "" {
			skipped++
			continue
		}
		valid = append(valid, parsed)
	}
	if len(valid) == 0 {
		return nil, format, fmt.Errorf("源返回 %d 行内容，未解析出有效节点（跳过 %d 行），请检查机场订阅地址是否正确", len(lines), skipped)
	}
	return valid, format, nil
}

func SyncAllAirports() {
	log.Println("[Cron] 开始每日凌晨3点的机场同步和测活任务...")
	airports, err := models.GetAirports()
	if err != nil {
		log.Println("[Cron] 获取机场列表失败:", err)
		return
	}
	for _, a := range airports {
		if err := SyncAirportNodeTask(a.ID); err != nil {
			log.Printf("[Cron] 机场 %s 同步失败: %v\n", a.Name, err)
		}
	}
	log.Println("[Cron] 每日机场同步任务已下发完毕。")
}

func SyncAirportNodeTask(airportID int) error {
	var a models.Airport
	a.ID = airportID
	if err := a.Find(); err != nil {
		log.Println("[Sync] 机场不存在 ID:", airportID)
		return err
	}

	log.Printf("[Sync] 开始同步机场: %s\n", a.Name)

	req, err := http.NewRequest("GET", a.URL, nil)
	if err != nil {
		log.Printf("[Sync] 机场 %s 请求构建失败: %v\n", a.Name, err)
		return err
	}
	req.Header.Set("User-Agent", "v2rayNG/1.8.5")
	client := utils.SafeHTTPClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Sync] 机场 %s 请求失败: %v\n", a.Name, err)
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	validNodes, format, parseErr := nodesFromSubscriptionBody(body)
	if parseErr != nil {
		log.Printf("[Sync] 机场 %s 解析失败(格式:%s): %v\n", a.Name, format, parseErr)
		return fmt.Errorf("机场 %s 解析失败(格式:%s): %w", a.Name, format, parseErr)
	}
	log.Printf("[Sync] 机场 %s 解析成功，格式:%s，节点数:%d\n", a.Name, format, len(validNodes))

	// 并发测活
	log.Printf("[Sync] 机场 %s 获取到 %d 个节点，开始并发测活 (AutoCleanup: %v)\n", a.Name, len(validNodes), a.AutoCleanup)

	var aliveNodes []models.Node
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20) // 最大并发数 20

	for _, n := range validNodes {
		wg.Add(1)
		go func(nd models.Node) {
			defer wg.Done()
			defer utils.RecoverPanic("airport-node-probe")
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if !a.AutoCleanup {
				// 不清理，则全部保留
				mu.Lock()
				aliveNodes = append(aliveNodes, nd)
				mu.Unlock()
				return
			}

			// 进行测活 (TCP Ping)
			host, port := node.ExtractServerHost(nd.Link)
			if host == "" || port == 0 {
				return
			}
			addr := host + ":" + strconv.Itoa(port)
			rtt := node.TCPPing(addr, 4*time.Second)

			// 存活判断
			isAlive := (rtt > 0)

			// 如果死了，但设置了专线免死牌，则强行保留
			if !isAlive && a.IsDedicated {
				isAlive = true
			}

			if isAlive {
				mu.Lock()
				aliveNodes = append(aliveNodes, nd)
				mu.Unlock()
			}
		}(n)
	}
	wg.Wait()

	log.Printf("[Sync] 机场 %s 测活完毕，最终存活/保留节点数: %d\n", a.Name, len(aliveNodes))

	if len(aliveNodes) == 0 {
		return fmt.Errorf("机场 %s 解析到 %d 个节点但全部测活失败，已保留原分组绑定，未清空", a.Name, len(validNodes))
	}

	// 事务内清空该分组并写入存活节点，避免清空成功但写入失败导致订阅整体失效
	err = models.DB.Transaction(func(tx *gorm.DB) error {
		var gn models.GroupNode
		tx.Where("name = ?", a.Name).First(&gn)
		if gn.ID != 0 {
			if err := tx.Model(&gn).Association("Nodes").Clear(); err != nil {
				return err
			}
		} else {
			gn = models.GroupNode{Name: a.Name}
			if err := tx.Create(&gn).Error; err != nil {
				return err
			}
		}
		for _, n := range aliveNodes {
			var dbNode models.Node
			tx.Where("name = ? AND link = ?", n.Name, n.Link).First(&dbNode)
			if dbNode.ID == 0 {
				if err := tx.Create(&n).Error; err != nil {
					return err
				}
				dbNode = n
			}
			if err := tx.Model(&dbNode).Association("GroupNodes").Append(&gn); err != nil {
				return err
			}
		}
		now := time.Now()
		a.LastSync = &now
		a.NodeCount = len(aliveNodes)
		return tx.Save(&a).Error
	})
	if err != nil {
		return fmt.Errorf("机场 %s 落库失败: %w", a.Name, err)
	}

	InvalidateOverview() // 机场同步增删节点，使概览缓存失效

	log.Printf("[Sync] 机场 %s 同步并落库完成。\n", a.Name)
	return nil
}
