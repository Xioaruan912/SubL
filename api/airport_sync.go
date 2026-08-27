package api

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"ppeelink/models"
	"ppeelink/node"
)

func SyncAllAirports() {
	log.Println("[Cron] 开始每日凌晨3点的机场同步和测活任务...")
	airports, err := models.GetAirports()
	if err != nil {
		log.Println("[Cron] 获取机场列表失败:", err)
		return
	}
	for _, a := range airports {
		SyncAirportNodeTask(a.ID)
	}
	log.Println("[Cron] 每日机场同步任务已下发完毕。")
}

func SyncAirportNodeTask(airportID int) {
	var a models.Airport
	a.ID = airportID
	if err := a.Find(); err != nil {
		log.Println("[Sync] 机场不存在 ID:", airportID)
		return
	}

	log.Printf("[Sync] 开始同步机场: %s, URL: %s\n", a.Name, a.URL)

	req, err := http.NewRequest("GET", a.URL, nil)
	if err != nil {
		log.Printf("[Sync] 机场 %s 请求构建失败: %v\n", a.Name, err)
		return
	}
	req.Header.Set("User-Agent", "v2rayNG/1.8.5")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Sync] 机场 %s 请求失败: %v\n", a.Name, err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// 解码并解析所有节点
	var urls []string
	decodedStr := node.Base64Decode(string(body))
	if decodedStr == string(body) || decodedStr == "" {
		// 没有被 Base64 decode，尝试原生按行切分
		urls = strings.Split(string(body), "\n")
	} else {
		urls = strings.Split(decodedStr, "\n")
	}

	var validNodes []models.Node
	for _, link := range urls {
		link = strings.TrimSpace(link)
		if link == "" || !strings.Contains(link, "://") {
			continue
		}
		n := models.Node{Link: link}
		n, err = DocodeNodeName(&n)
		if err != nil || n.Name == "" {
			continue
		}
		validNodes = append(validNodes, n)
	}

	if len(validNodes) == 0 {
		log.Printf("[Sync] 机场 %s 未获取到有效节点\n", a.Name)
		return
	}

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

	// 清理该分组原本的所有节点
	var gn models.GroupNode
	models.DB.Where("name = ?", a.Name).First(&gn)
	if gn.ID != 0 {
		models.DB.Model(&gn).Association("Nodes").Clear()
	} else {
		gn = models.GroupNode{Name: a.Name}
		gn.Add()
	}

	// 存入存活节点，并绑定分组
	for _, n := range aliveNodes {
		var dbNode models.Node
		models.DB.Where("name = ? AND link = ?", n.Name, n.Link).First(&dbNode)
		if dbNode.ID == 0 {
			n.Add()
			models.DB.Where("name = ? AND link = ?", n.Name, n.Link).First(&dbNode)
		}
		if dbNode.ID != 0 {
			dbNode.UpdateGroup([]models.GroupNode{{Name: a.Name}})
		}
	}

	// 更新机场的最新状态
	now := time.Now()
	a.LastSync = &now
	a.NodeCount = len(aliveNodes)
	a.Update()

	log.Printf("[Sync] 机场 %s 同步并落库完成。\n", a.Name)
}
