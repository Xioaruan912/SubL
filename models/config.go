package models

import (
	"fmt"
	"log"
	"os"
	"ppeelink/utils"

	"gopkg.in/yaml.v3"
)

// type Config struct {
// 	ID    int
// 	Key   string
// 	Value string
// }

// Config 配置结构体
type Config struct {
	JwtSecret  string `yaml:"jwt_secret"`  // JWT密钥
	ExpireDays int    `yaml:"expire_days"` // 过期天数
	Port       int    `yaml:"port"`        // 端口号
}

var comment string = `# jwt_secret: JWT密钥
# expire_days: token 过期天数
# port: 启动端口
`

// 初始化配置
func ConfigInit() {
	// 确保配置目录存在，否则首次安装时配置无法写入
	if err := os.MkdirAll("./db", 0o755); err != nil {
		log.Println("创建配置目录失败:", err)
	}
	_, statErr := os.Stat("./db/config.yaml")
	missing := os.IsNotExist(statErr)
	cfg := ReadConfig()
	dirty := missing
	if cfg.JwtSecret == "" {
		cfg.JwtSecret = utils.RandString(43) // 加密安全随机JWT密钥
		dirty = true
	}
	if cfg.ExpireDays == 0 {
		cfg.ExpireDays = 14
		dirty = true
	}
	if cfg.Port == 0 {
		cfg.Port = 8000
		dirty = true
	}
	if dirty {
		if err := writeConfigFile(cfg); err != nil {
			fmt.Println("写入文件失败:", err)
			return
		}
		if missing {
			log.Println("配置文件不存在，已创建默认配置文件")
		} else {
			log.Println("已补全配置文件缺失字段")
		}
	}
}

func writeConfigFile(cfg Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	data = []byte(comment + string(data)) // 添加注释
	return os.WriteFile("./db/config.yaml", data, 0600)
}

// 读取配置
func ReadConfig() Config {
	file, err := os.ReadFile("./db/config.yaml")
	if err != nil {
		log.Println(err)
	}
	cfg := Config{}
	yaml.Unmarshal(file, &cfg)
	return cfg
}

// 设置配置
func SetConfig(newCfg Config) {
	oldCfg := ReadConfig() // 读取旧的配置文件
	// 覆盖新的字段
	if newCfg.JwtSecret != "" {
		oldCfg.JwtSecret = newCfg.JwtSecret
	}
	if newCfg.ExpireDays != 0 {
		oldCfg.ExpireDays = newCfg.ExpireDays
	}
	if newCfg.Port != 0 {
		oldCfg.Port = newCfg.Port
	}
	if err := writeConfigFile(oldCfg); err != nil {
		log.Println("写入配置文件失败:", err)
	}
}
