// api/subcription.go

package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ppeelink/models"
	"ppeelink/node"

	"github.com/gin-gonic/gin"
)

// fetchSubscriptionSource supports newline/comma separated fallback sources.
// The first successful 2xx response wins, so one failed airport endpoint no
// longer prevents a subscription from being refreshed.
func fetchSubscriptionSource(raw string) ([]byte, string, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == ',' })
	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error
	for _, source := range parts {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		req, err := http.NewRequest("GET", source, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "v2rayNG/1.8.5")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("%s 返回 HTTP %d", source, resp.StatusCode)
			continue
		}
		if len(body) == 0 {
			lastErr = fmt.Errorf("%s 返回空内容", source)
			continue
		}
		return body, source, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("没有可用的订阅源")
	}
	return nil, "", lastErr
}

// ensureAirportFromSub 订阅从外部机场URL导入节点后，自动创建/复用同名机场记录
func ensureAirportFromSub(name, url string, nodeCount int) {
	now := time.Now()
	var ap models.Airport
	err := models.DB.Where("name = ?", name).First(&ap).Error
	if err == nil {
		// 复用更新
		models.DB.Model(&ap).Updates(map[string]interface{}{
			"url":        url,
			"node_count": nodeCount,
			"last_sync":  now,
		})
		return
	}
	// 新建
	models.DB.Create(&models.Airport{
		Name:      name,
		URL:       url,
		LastSync:  &now,
		NodeCount: nodeCount,
	})
}

// parseGroups 解析逗号分隔的分组 ID 参数为 GroupNode 引用列表
func parseGroups(groups string) ([]models.GroupNode, error) {
	var refs []models.GroupNode
	if groups == "" {
		return refs, nil
	}
	for _, g := range strings.Split(groups, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(g))
		if err != nil || id == 0 {
			continue
		}
		var gn models.GroupNode
		if err := models.DB.Where("id = ?", id).First(&gn).Error; err != nil {
			return nil, err
		}
		refs = append(refs, models.GroupNode{ID: gn.ID})
	}
	return refs, nil
}

func SubTotal(c *gin.Context) {
	var Sub models.Subcription
	subs, err := Sub.List()
	count := len(subs)
	if err != nil {
		c.JSON(500, gin.H{
			"msg": "取得订阅总数失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": count,
		"msg":  "取得订阅总数",
	})
}

// 获取订阅列表
func SubGet(c *gin.Context) {
	var Sub models.Subcription
	Subs, err := Sub.List()
	if err != nil {
		c.JSON(500, gin.H{
			"msg": "node list error",
		})
		return
	}
	// 加载每个订阅引用的分组（含节点数），供前端回显
	for i := range Subs {
		models.DB.Preload("GroupRefs").First(&Subs[i], Subs[i].ID)
		for j := range Subs[i].GroupRefs {
			var cnt int64
			models.DB.Table("group_node_nodes").Where("group_node_id = ?", Subs[i].GroupRefs[j].ID).Count(&cnt)
			Subs[i].GroupRefs[j].NodeCount = int(cnt)
		}
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": Subs,
		"msg":  "node get",
	})
}

// SubPreviewNodes 预览合并后的订阅节点（不记录日志、不下发配置）
func SubPreviewNodes(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "id 不能为空"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "无效的 ID"})
		return
	}
	var sub models.Subcription
	if err := models.DB.First(&sub, id).Error; err != nil {
		c.JSON(404, gin.H{"code": "40400", "msg": "订阅不存在"})
		return
	}
	if err := mergeGroupNodes(&sub); err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "合并节点失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": sub.Nodes,
		"msg":  "节点预览",
	})
}

// 添加订阅
func SubAdd(c *gin.Context) {
	name := c.PostForm("name")
	configs := c.PostForm("config")
	nodes := c.PostForm("nodes")
	airportUrl := c.PostForm("airport_url")
	groups := c.PostForm("groups")
	pipeline := c.PostForm("pipeline")

	if name == "" {
		c.JSON(400, gin.H{
			"msg": "订阅名称不能为空",
		})
		return
	}
	if nodes == "" && airportUrl == "" && groups == "" {
		c.JSON(400, gin.H{
			"msg": "必须提供节点列表、机场订阅链接或分组引用",
		})
		return
	}

	groupRefs, err := parseGroups(groups)
	if err != nil {
		c.JSON(400, gin.H{"msg": "分组引用无效: " + err.Error()})
		return
	}

	var NodesData []models.Node
	var nodeNames []string
	var lastSyncAt *time.Time

	// 如果提供了机场URL，优先从机场获取节点
	if airportUrl != "" {
		body, activeURL, err := fetchSubscriptionSource(airportUrl)
		if err != nil {
			c.JSON(400, gin.H{"msg": "请求订阅链接失败: " + err.Error()})
			return
		}
		now := time.Now()
		lastSyncAt = &now

		// 尝试 Base64 解码，绝大多数标准订阅都是 Base64 编码的链接列表
		decodedStr := node.Base64Decode(string(body))
		links := strings.Split(decodedStr, "\n")

		for _, link := range links {
			link = strings.TrimSpace(link)
			if link == "" || !strings.Contains(link, "://") {
				continue
			}
			n := models.Node{Link: link}
			n, err = DocodeNodeName(&n) // 解析节点名称
			if err != nil || n.Name == "" {
				continue
			}
			// 保存到数据库
			n.Add()

			// 为了防止重名导致获取错节点，我们通过精确查找获取 ID
			var dbNode models.Node
			models.DB.Model(models.Node{}).Where("name = ? AND link = ?", n.Name, n.Link).First(&dbNode)
			if dbNode.ID != 0 {
				NodesData = append(NodesData, dbNode)
				nodeNames = append(nodeNames, dbNode.Name)

				// 自动将机场抓取到的节点归类到一个以订阅名称命名的分组，防止在节点列表中过于杂乱
				gn := models.GroupNode{Name: name}
				gn.Add()
				dbNode.UpdateGroup([]models.GroupNode{{Name: name}})
			}
		}
		// 自动创建/复用同名机场记录，使其出现在机场管理
		ensureAirportFromSub(name, activeURL, len(nodeNames))
	} else {
		// 常规的手动选择节点
		for _, nodeName := range strings.Split(nodes, ",") {
			if strings.TrimSpace(nodeName) == "" {
				continue
			}
			FirstNode := models.Node{
				Name: nodeName,
			}
			result := models.DB.Model(models.Node{}).Where("name = ?", FirstNode.Name).First(&FirstNode)
			if result.Error != nil {
				c.JSON(400, gin.H{"msg": "节点不存在: " + result.Error.Error()})
				return
			}
			NodesData = append(NodesData, FirstNode)
			nodeNames = append(nodeNames, FirstNode.Name)
		}
	}

	sub := models.Subcription{
		Name:             name,
		Config:           configs,
		NodeOrder:        strings.Join(nodeNames, ","),
		Nodes:            NodesData,
		GroupRefs:        groupRefs,
		Pipeline:         pipeline,
		SourceURLs:       airportUrl,
		LastGoodSnapshot: strings.Join(nodeNames, "\n"),
		LastSyncAt:       lastSyncAt,
	}
	err = sub.Add()
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "添加订阅失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "添加订阅成功",
	})
}

// 更新订阅
func SubUpdate(c *gin.Context) {
	NewName := c.PostForm("name")
	OldName := c.PostForm("oldname")
	configs := c.PostForm("config")
	nodes := c.PostForm("nodes")
	airportUrl := c.PostForm("airport_url")
	groups := c.PostForm("groups")
	pipeline := c.PostForm("pipeline")

	if NewName == "" {
		c.JSON(400, gin.H{
			"msg": "订阅名称不能为空",
		})
		return
	}
	if nodes == "" && airportUrl == "" && groups == "" {
		c.JSON(400, gin.H{
			"msg": "必须提供节点列表、机场订阅链接或分组引用",
		})
		return
	}

	groupRefs, err := parseGroups(groups)
	if err != nil {
		c.JSON(400, gin.H{"msg": "分组引用无效: " + err.Error()})
		return
	}

	var NodesData []models.Node
	var nodeNames []string
	var lastSyncAt *time.Time
	var lastSyncError string

	if airportUrl != "" {
		body, activeURL, err := fetchSubscriptionSource(airportUrl)
		if err != nil {
			// Keep serving the last known-good nodes when every source is down.
			var existing models.Subcription
			if dbErr := models.DB.Preload("Nodes").Where("name = ?", OldName).First(&existing).Error; dbErr != nil || len(existing.Nodes) == 0 {
				c.JSON(400, gin.H{"msg": "请求订阅链接失败且无可用缓存: " + err.Error()})
				return
			}
			NodesData = append(NodesData, existing.Nodes...)
			for _, cached := range existing.Nodes {
				nodeNames = append(nodeNames, cached.Name)
			}
			lastSyncError = err.Error()
		} else {
			now := time.Now()
			lastSyncAt = &now
		}

		decodedStr := ""
		if len(body) > 0 {
			decodedStr = node.Base64Decode(string(body))
		}
		links := strings.Split(decodedStr, "\n")

		for _, link := range links {
			link = strings.TrimSpace(link)
			if link == "" || !strings.Contains(link, "://") {
				continue
			}
			n := models.Node{Link: link}
			n, err = DocodeNodeName(&n)
			if err != nil || n.Name == "" {
				continue
			}
			n.Add()

			var dbNode models.Node
			models.DB.Model(models.Node{}).Where("name = ? AND link = ?", n.Name, n.Link).First(&dbNode)
			if dbNode.ID != 0 {
				NodesData = append(NodesData, dbNode)
				nodeNames = append(nodeNames, dbNode.Name)

				gn := models.GroupNode{Name: NewName}
				gn.Add()
				dbNode.UpdateGroup([]models.GroupNode{{Name: NewName}})
			}
		}
		// 自动创建/复用同名机场记录，使其出现在机场管理
		if activeURL != "" {
			ensureAirportFromSub(NewName, activeURL, len(nodeNames))
		}
	} else {
		for _, nodeName := range strings.Split(nodes, ",") {
			if strings.TrimSpace(nodeName) == "" {
				continue
			}
			FirstNode := models.Node{
				Name: nodeName,
			}
			result := models.DB.Model(models.Node{}).Where("name = ?", FirstNode.Name).First(&FirstNode)
			if result.Error != nil {
				c.JSON(400, gin.H{"msg": result.Error.Error()})
				return
			}
			NodesData = append(NodesData, FirstNode)
			nodeNames = append(nodeNames, FirstNode.Name)
		}
	}

	OldSub := models.Subcription{Name: OldName}
	NewSub := models.Subcription{
		Name:             NewName,
		Config:           configs,
		NodeOrder:        strings.Join(nodeNames, ","),
		Nodes:            NodesData,
		GroupRefs:        groupRefs,
		Pipeline:         pipeline,
		SourceURLs:       airportUrl,
		LastGoodSnapshot: strings.Join(nodeNames, "\n"),
		LastSyncAt:       lastSyncAt,
		LastSyncError:    lastSyncError,
	}

	err = OldSub.Update(&NewSub)
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "更新订阅失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "更新订阅成功",
	})
}

// 删除订阅 (无需修改)
func SubDel(c *gin.Context) {
	var sub models.Subcription
	id := c.Query("id")
	if id == "" {
		c.JSON(400, gin.H{
			"msg": "id 不能为空",
		})
		return
	}
	x, err := strconv.Atoi(id) // 增加错误检查
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "无效的 ID: " + err.Error(),
		})
		return
	}
	sub.ID = x
	err = sub.Find()
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "查找订阅失败: " + err.Error(),
		})
		return
	}
	err = sub.Del()
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "删除订阅失败: " + err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "删除订阅成功",
	})
}

// 重置订阅令牌（重置后旧订阅链接立即失效）
// POST /api/v1/subcription/reset-token  body: id
func ResetSubToken(c *gin.Context) {
	idStr := c.PostForm("id")
	if idStr == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "id 不能为空"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "无效的 ID"})
		return
	}
	sub := models.Subcription{ID: id}
	if err := sub.Find(); err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "查找订阅失败: " + err.Error()})
		return
	}
	newToken := models.GenerateToken()
	if err := models.DB.Model(&sub).Update("token", newToken).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "重置令牌失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"token": newToken}, "msg": "订阅链接已重置"})
}

// 设置/清除订阅过期时间
// POST /api/v1/subcription/set-expire  body: id + expire(Unix秒, 留空=永不过期)
func SetSubExpire(c *gin.Context) {
	idStr := c.PostForm("id")
	if idStr == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "id 不能为空"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "无效的 ID"})
		return
	}
	sub := models.Subcription{ID: id}
	if err := sub.Find(); err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "查找订阅失败: " + err.Error()})
		return
	}
	expireStr := c.PostForm("expire")
	var expiresAt *time.Time
	if expireStr != "" {
		ts, err := strconv.ParseInt(expireStr, 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"code": "40000", "msg": "过期时间格式错误"})
			return
		}
		t := time.Unix(ts, 0)
		expiresAt = &t
	}
	if err := models.DB.Model(&sub).Update("expires_at", expiresAt).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "设置过期失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "过期时间已更新"})
}
