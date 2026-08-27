package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"ppeelink/models"
	"ppeelink/node"
	"time"

	"github.com/gin-gonic/gin"
)

// md5加密
func Md5(src string) string {
	m := md5.New()
	m.Write([]byte(src))
	res := hex.EncodeToString(m.Sum(nil))
	return res
}

// mergeGroupNodes 将订阅引用的分组节点合并进 sub.Nodes（去重，分组节点在后）
// 分组引用按 ID 关联，机场重同步后分组节点更新，订阅拉取时自动跟进
func mergeGroupNodes(sub *models.Subcription) error {
	if sub.ID == 0 {
		return nil
	}
	var withRefs models.Subcription
	if err := models.DB.Preload("Nodes").Preload("GroupRefs.Nodes").First(&withRefs, sub.ID).Error; err != nil {
		return err
	}
	// 收集当前手动节点（按名称去重）
	seen := map[string]bool{}
	merged := []models.Node{}
	for _, n := range withRefs.Nodes {
		if n.Name == "" || seen[n.Name] {
			continue
		}
		seen[n.Name] = true
		merged = append(merged, n)
	}
	// 追加各分组当前节点
	for _, g := range withRefs.GroupRefs {
		for _, n := range g.Nodes {
			if n.Name == "" || seen[n.Name] {
				continue
			}
			seen[n.Name] = true
			merged = append(merged, n)
		}
	}

	// 订阅名匹配机场且机场勾选了节点时，统一按勾选过滤（覆盖手动 Nodes 与分组节点）
	var ap models.Airport
	if err := models.DB.Where("name = ?", sub.Name).First(&ap).Error; err == nil && ap.SelectedNodes != "" {
		selectedSet := map[string]bool{}
		for _, s := range strings.Split(ap.SelectedNodes, ",") {
			if s = strings.TrimSpace(s); s != "" {
				selectedSet[s] = true
			}
		}
		filtered := merged[:0]
		for _, n := range merged {
			if selectedSet[n.Name] {
				filtered = append(filtered, n)
			}
		}
		merged = filtered
	}

	sub.Nodes = merged
	return nil
}

// subName 从请求上下文取订阅名（由 GetClient 匹配后设置，避免全局变量并发串号）
func subName(c *gin.Context) string {
	v, _ := c.Get("subname")
	s, _ := v.(string)
	return s
}

func GetClient(c *gin.Context) {
	// 获取协议头
	token := c.Query("token")
	ClientIndex := c.Query("client") // 客户端标识
	if token == "" {
		log.Println("token为空")
		c.Writer.WriteString("token为空")
		return
	}
	Sub := new(models.Subcription)
	// 获取所有订阅
	list, _ := Sub.List()
	// 按令牌匹配：优先随机 token，兼容旧的 md5(订阅名) 链接
	for _, sub := range list {
		matched := false
		if sub.Token != "" && strings.EqualFold(sub.Token, token) {
			matched = true
		} else if strings.ToLower(Md5(sub.Name)) == strings.ToLower(token) {
			matched = true
		}
		if !matched {
			continue
		}
		// 过期校验
		if sub.ExpiresAt != nil && time.Now().After(*sub.ExpiresAt) {
			c.Writer.WriteString("订阅已过期")
			return
		}
		// 记录订阅名供后续子函数使用
		c.Set("subname", sub.Name)
		// 判断是否带客户端参数
		switch ClientIndex {
		case "clash":
			GetClash(c)
			return
		case "surge":
			GetSurge(c)
			return
		case "loon":
			GetLoon(c)
			return
		case "v2ray":
			GetV2ray(c)
			return
		}
		// 自动识别客户端
		ClientList := []string{"clash", "surge", "loon"}
		for k, v := range c.Request.Header {
			if k == "User-Agent" {
				for _, UserAgent := range v {
					for _, client := range ClientList {
						if strings.Contains(strings.ToLower(UserAgent), strings.ToLower(client)) {
							switch client {
							case "clash":
								GetClash(c)
								return
							case "surge":
								GetSurge(c)
								return
							case "loon":
								GetLoon(c)
								return
							}
						}
					}
					GetV2ray(c)
				}
			}
		}
		return
	}
	c.Writer.WriteString("无效的订阅令牌")
}
func GetV2ray(c *gin.Context) {
	var sub models.Subcription
	if subName(c) == "" {
		c.Writer.WriteString("订阅名为空")
		return
	}
	// subname := c.Param("subname")
	// subname := SunName
	// subname = node.Base64Decode(subname)
	sub.Name = subName(c)
	err := sub.Find()
	if err != nil {
		c.Writer.WriteString("找不到这个订阅:" + subName(c))
		return
	}
	// 合并引用分组节点（机场同步自动跟进）
	if err := mergeGroupNodes(&sub); err != nil {
		log.Println("合并分组节点失败:", err)
	}
	baselist := ""
	for _, v := range sub.Nodes {
		switch {
		// 如果包含多条节点
		case strings.Contains(v.Link, ","):
			links := strings.Split(v.Link, ",")
			baselist += strings.Join(links, "\n") + "\n"
			continue
		//如果是订阅转换
		case strings.Contains(v.Link, "http://") || strings.Contains(v.Link, "https://"):
			resp, err := http.Get(v.Link)
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			nodes := node.Base64Decode(string(body))
			baselist += nodes + "\n"
		// 默认
		default:
			baselist += v.Link + "\n"
		}
	}
	c.Set("subname", subName(c))
	filename := fmt.Sprintf("%s.txt", subName(c))
	encodedFilename := url.QueryEscape(filename)
	c.Writer.Header().Set("Content-Disposition", "inline; filename*=utf-8''"+encodedFilename)
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteString(node.Base64Encode(baselist))
}
func GetClash(c *gin.Context) {
	var sub models.Subcription
	// subname := c.Param("subname")
	// subname := node.Base64Decode(SunName)
	sub.Name = subName(c)
	err := sub.Find()
	if err != nil {
		c.Writer.WriteString("找不到这个订阅:" + subName(c))
		return
	}
	// 合并引用分组节点（机场同步自动跟进）
	if err := mergeGroupNodes(&sub); err != nil {
		log.Println("合并分组节点失败:", err)
	}

	urls := []string{}

	log.Println("订阅名:", sub.Nodes)
	for _, v := range sub.Nodes {
		log.Println("节点信息:", v)
		log.Println("节点链接:", v.Link)
		switch {
		// 如果包含多条节点
		case strings.Contains(v.Link, ","):
			links := strings.Split(v.Link, ",")
			urls = append(urls, links...)
			continue
		//如果是订阅转换
		case strings.Contains(v.Link, "http://") || strings.Contains(v.Link, "https://"):
			resp, err := http.Get(v.Link)
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			nodes := node.Base64Decode(string(body))
			links := strings.Split(nodes, "\n")
			urls = append(urls, links...)
		// 默认
		default:
			urls = append(urls, v.Link)
		}
	}
	log.Println("urls", urls)
	var configs node.SqlConfig
	err = json.Unmarshal([]byte(sub.Config), &configs)
	if err != nil {
		c.Writer.WriteString("配置读取错误")
		return
	}
	DecodeClash, err := node.EncodeClash(urls, configs)
	if err != nil {
		c.Writer.WriteString(err.Error())
		return
	}
	c.Set("subname", subName(c))
	filename := fmt.Sprintf("%s.yaml", subName(c))
	encodedFilename := url.QueryEscape(filename)
	c.Writer.Header().Set("Content-Disposition", "inline; filename*=utf-8''"+encodedFilename)
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteString(string(DecodeClash))
}
func GetSurge(c *gin.Context) {
	var sub models.Subcription
	// subname := c.Param("subname")
	// subname := node.Base64Decode(SunName)
	sub.Name = subName(c)
	err := sub.Find()
	if err != nil {
		c.Writer.WriteString("找不到这个订阅:" + subName(c))
		return
	}
	// 合并引用分组节点（机场同步自动跟进）
	if err := mergeGroupNodes(&sub); err != nil {
		log.Println("合并分组节点失败:", err)
	}
	urls := []string{}
	for _, v := range sub.Nodes {
		switch {
		// 如果包含多条节点
		case strings.Contains(v.Link, ","):
			links := strings.Split(v.Link, ",")
			urls = append(urls, links...)
			continue
		//如果是订阅转换
		case strings.Contains(v.Link, "http://") || strings.Contains(v.Link, "https://"):
			resp, err := http.Get(v.Link)
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			nodes := node.Base64Decode(string(body))
			links := strings.Split(nodes, "\n")
			urls = append(urls, links...)
		// 默认
		default:
			urls = append(urls, v.Link)
		}
	}

	var configs node.SqlConfig
	err = json.Unmarshal([]byte(sub.Config), &configs)
	if err != nil {
		c.Writer.WriteString("配置读取错误")
		return
	}
	// log.Println("surge路径:", configs)
	DecodeClash, err := node.EncodeSurge(urls, configs)
	if err != nil {
		c.Writer.WriteString(err.Error())
		return
	}
	c.Set("subname", subName(c))
	filename := fmt.Sprintf("%s.conf", subName(c))
	encodedFilename := url.QueryEscape(filename)
	c.Writer.Header().Set("Content-Disposition", "inline; filename*=utf-8''"+encodedFilename)
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	host := c.Request.Host
	url := c.Request.URL.String()
	// 如果包含头部更新信息
	if strings.Contains(DecodeClash, "#!MANAGED-CONFIG") {
		c.Writer.WriteString(DecodeClash)
		return
	}
	// 否则就插入头部更新信息
	interval := fmt.Sprintf("#!MANAGED-CONFIG %s interval=86400 strict=false", host+url)
	c.Writer.WriteString(string(interval + "\n" + DecodeClash))
}

// GetLoon 输出 Loon 订阅（填充 [Proxy] 段，策略组靠 Remote Filter 筛选）
func GetLoon(c *gin.Context) {
	var sub models.Subcription
	sub.Name = subName(c)
	err := sub.Find()
	if err != nil {
		c.Writer.WriteString("找不到这个订阅:" + subName(c))
		return
	}
	// 合并引用分组节点（机场同步自动跟进）
	if err := mergeGroupNodes(&sub); err != nil {
		log.Println("合并分组节点失败:", err)
	}

	urls := []string{}
	for _, v := range sub.Nodes {
		switch {
		// 如果包含多条节点
		case strings.Contains(v.Link, ","):
			links := strings.Split(v.Link, ",")
			urls = append(urls, links...)
			continue
		// 如果是订阅转换
		case strings.Contains(v.Link, "http://") || strings.Contains(v.Link, "https://"):
			resp, err := http.Get(v.Link)
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			nodes := node.Base64Decode(string(body))
			links := strings.Split(nodes, "\n")
			urls = append(urls, links...)
		// 默认
		default:
			urls = append(urls, v.Link)
		}
	}

	var configs node.SqlConfig
	err = json.Unmarshal([]byte(sub.Config), &configs)
	if err != nil {
		c.Writer.WriteString("配置读取错误")
		return
	}
	loonText, err := node.EncodeLoon(urls, configs)
	if err != nil {
		c.Writer.WriteString(err.Error())
		return
	}
	c.Set("subname", subName(c))
	filename := fmt.Sprintf("%s.conf", subName(c))
	encodedFilename := url.QueryEscape(filename)
	c.Writer.Header().Set("Content-Disposition", "inline; filename*=utf-8''"+encodedFilename)
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteString(loonText)
}
