package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"ppeelink/client"
	"ppeelink/middlewares"
	"ppeelink/models"
	"ppeelink/routers"
	"ppeelink/settings"
	"ppeelink/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// Include Vite-generated assets whose names may begin with "_" or ".".
// The default embed matcher excludes those files when walking directories,
// which can otherwise produce runtime 404s for chunks such as
// _plugin-vue_export-helper.*.js.
//go:embed all:webs/dist/*
var embeddedFiles embed.FS

//go:embed template
var Template embed.FS

// 版本号
var version string

func Templateinit() {
	// 设置template路径
	// 检查目录是否创建
	subFS, err := fs.Sub(Template, "template")
	if err != nil {
		log.Println(err)
		return // 如果出错，直接返回
	}
	entries, err := fs.ReadDir(subFS, ".")
	if err != nil {
		log.Println(err)
		return // 如果出错，直接返回
	}
	// 创建template目录
	_, err = os.Stat("./template")
	if os.IsNotExist(err) {
		err = os.Mkdir("./template", 0666)
		if err != nil {
			log.Println(err)
			return
		}
	}
	// 写入默认模板
	for _, entry := range entries {
		_, err := os.Stat("./template/" + entry.Name())
		//如果文件不存在则写入默认模板
		if os.IsNotExist(err) {
			data, err := fs.ReadFile(subFS, entry.Name())
			if err != nil {
				log.Println(err)
				continue
			}
			err = os.WriteFile("./template/"+entry.Name(), data, 0666)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func main() {
	// 初始化配置
	models.ConfigInit()
	config := models.ReadConfig() // 读取配置文件
	var port = config.Port        // 读取端口号
	// 获取版本号
	var Isversion bool
	version = "2.1"
	flag.BoolVar(&Isversion, "version", false, "显示版本号")
	flag.Parse()
	if Isversion {
		fmt.Println(version)
		return
	}
	// 初始化数据库
	models.InitSqlite()
	// 获取命令行参数
	args := os.Args
	// 如果长度小于2则没有接收到任何参数
	if len(args) < 2 {
		Run(port)
		return
	}
	// 命令行参数选择
	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)
	var username, password string
	settingCmd.StringVar(&username, "username", "", "设置账号")
	settingCmd.StringVar(&password, "password", "", "设置密码")
	settingCmd.IntVar(&port, "port", 8000, "修改端口")
	switch args[1] {
	// 解析setting命令标志
	case "setting":
		settingCmd.Parse(args[2:])
		fmt.Println(username, password)
		settings.ResetUser(username, password)
		return
	case "run":
		settingCmd.Parse(args[2:])
		models.SetConfig(models.Config{
			Port: port,
		}) // 设置端口
		Run(port)
	default:
		return

	}
}

func Run(port int) {
	// 初始化gin框架
	r := gin.Default()
	// Do not trust arbitrary X-Forwarded-For by default. Operators behind a
	// reverse proxy can explicitly set SUBLINKX_TRUSTED_PROXIES to a
	// comma-separated list of proxy IPs/CIDRs.
	trusted := []string{}
	for _, item := range strings.Split(os.Getenv("SUBLINKX_TRUSTED_PROXIES"), ",") {
		if value := strings.TrimSpace(item); value != "" {
			trusted = append(trusted, value)
		}
	}
	if err := r.SetTrustedProxies(trusted); err != nil {
		log.Printf("可信代理配置无效: %v", err)
	}
	// 初始化日志配置
	utils.Loginit()
	// 初始化模板
	Templateinit()
	// 安装中间件
	r.Use(middlewares.AuthorToken) // jwt验证token
	r.Use(middlewares.AuditTrail)  // 管理写操作审计（不记录请求体）
	// 设置静态资源路径（自定义 handler，支持 gzip + 正确 Content-Length）
	staticFiles, err := fs.Sub(embeddedFiles, "webs/dist")
	if err != nil {
		log.Println(err)
	}
	r.Any("/static/*filepath", middlewares.StaticFS(staticFiles))
	// 设置模板路径
	r.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(staticFiles, "index.html")
		if err != nil {
			c.Error(err)
			return

		}
		c.Data(200, "text/html", data)
	})
	// 注册路由
	routers.User(r)
	routers.Mentus(r)
	routers.Subcription(r)
	routers.Nodes(r)
	routers.Clients(r)
	routers.Total(r)
	routers.Templates(r)
	routers.Version(r, version)
	routers.Downloads(r)
	routers.AirportRoutes(r) // 注册机场管理路由
	routers.Rules(r)         // 规则中心
	routers.Tasks(r)         // 后台任务中心
	routers.Tokens(r)        // API Token 管理
	routers.Ops(r)           // 安全运维/备份
	routers.Status(r)        // 公共只读状态页
	routers.Regression(r)    // 分流回归用例与模板命中差异
	routers.Audit(r)         // 管理操作审计
	// 客户端下载目录 + 定时检查更新（启动即查 + 每 24h）
	os.MkdirAll("downloads", 0o755)
	client.Start()

	// 启动后台定时测活任务 (Cron)
	StartCronTasks()

	// 启动服务
	r.Run(fmt.Sprintf("0.0.0.0:%d", port))
}
