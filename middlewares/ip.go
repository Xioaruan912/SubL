package middlewares

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"ppeelink/models"
	"ppeelink/utils"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// GetIp 记录订阅访问者 IP。归属地查询与落库放到后台执行，避免拖慢订阅下载。
func GetIp(c *gin.Context) {
	c.Next()
	subname, ok := c.Get("subname")
	if !ok {
		return
	}
	name, ok := subname.(string)
	if !ok || name == "" {
		log.Println("无法获取订阅名称")
		return
	}
	go recordSubscriberIP(c.ClientIP(), name)
}

func recordSubscriberIP(ip, subname string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("记录订阅IP异常:", r)
		}
	}()

	client := utils.SafeHTTPClient(5 * time.Second)
	resp, err := client.Get(fmt.Sprintf("https://whois.pconline.com.cn/ipJson.jsp?ip=%s&json=true", ip))
	if err != nil {
		log.Println("获取IP信息失败:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	utf8Body, _ := simplifiedchinese.GBK.NewDecoder().Bytes(body)
	type IpInfo struct {
		Addr string `json:"addr"`
		Ip   string `json:"ip"`
	}
	ipinfo := IpInfo{}
	if err := json.Unmarshal(utf8Body, &ipinfo); err != nil {
		log.Println("解析IP信息失败:", err)
		return
	}

	var sub models.Subcription
	sub.Name = subname
	if err := sub.Find(); err != nil {
		log.Println("查找订阅失败:", err)
		return
	}

	var iplog models.SubLogs
	iplog.IP = ip
	if err := iplog.Find(sub.ID); err != nil {
		newIplog := models.SubLogs{
			IP:            ip,
			Addr:          ipinfo.Addr,
			SubcriptionID: sub.ID,
			Date:          time.Now().Format("2006-01-02 15:04:05"),
			Count:         1,
		}
		if err := newIplog.Add(); err != nil {
			log.Println("添加IP日志记录失败:", err)
		}
		return
	}
	iplog.Count++
	iplog.Date = time.Now().Format("2006-01-02 15:04:05")
	if err := iplog.Update(); err != nil {
		log.Println("更新IP日志记录失败:", err)
	}
}
