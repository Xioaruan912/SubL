package api

import (
	"github.com/gin-gonic/gin"
)

type Meta struct {
	Title     string   `json:"title"`
	Icon      string   `json:"icon"`
	Hidden    bool     `json:"hidden"`
	Roles     []string `json:"roles"`
	KeepAlive bool     `json:"keepAlive,omitempty"`
}

type Child struct {
	Path      string `json:"path"`
	Component string `json:"component"`
	Name      string `json:"name"`
	Meta      Meta   `json:"meta"`
}

type Menu struct {
	Path      string  `json:"path"`
	Component string  `json:"component"`
	Redirect  string  `json:"redirect"`
	Name      string  `json:"name"`
	Meta      Meta    `json:"meta"`
	Children  []Child `json:"children"`
}

func GetMenus(c *gin.Context) {
	menus := []Menu{
		{
			Path:      "/system",
			Component: "Layout",
			// Redirect:  "/system/user",
			Name: "system",
			Meta: Meta{
				Title:  "system",
				Icon:   "system",
				Hidden: true,
				Roles:  []string{"ADMIN"},
			},
			Children: []Child{
				{
					Path:      "user/set",
					Component: "system/user/set",
					Name:      "Userset",
					Meta: Meta{
						Title:     "userset",
						Icon:      "role",
						Hidden:    true,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
			},
		},
		// 订阅管理
		{
			Path:      "/subcription",
			Component: "Layout",
			Redirect:  "/subcription/subs",
			Name:      "subcription",
			Meta: Meta{
				Title:  "subcription",
				Icon:   "client",
				Hidden: false,
				Roles:  []string{"ADMIN"},
			},
			Children: []Child{
				{
					Path:      "subs",
					Component: "subcription/subs",
					Name:      "Subs",
					Meta: Meta{
						Title:     "sublist",
						Icon:      "link",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
				{
					Path:      "nodes",
					Component: "subcription/nodes",
					Name:      "Nodes",
					Meta: Meta{
						Title:     "nodelist",
						Icon:      "publish",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
				{
					Path:      "airport",
					Component: "subcription/airport",
					Name:      "Airport",
					Meta: Meta{
						Title:     "机场管理",
						Icon:      "upload",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
				{
					Path:      "clients",
					Component: "subcription/clients",
					Name:      "Clients",
					Meta: Meta{
						Title:     "clientdownload",
						Icon:      "download",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
			},
		},
		// 测试
		{
			Path:      "/test",
			Component: "Layout",
			Redirect:  "/test/unlock",
			Name:      "test",
			Meta: Meta{
				Title:  "test",
				Icon:   "monitor",
				Hidden: false,
				Roles:  []string{"ADMIN"},
			},
			Children: []Child{
				{
					Path:      "egress",
					Component: "test/egress",
					Name:      "EgressTest",
					Meta: Meta{
						Title:     "分流检测",
						Icon:      "connection",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
				{
					Path:      "unlock",
					Component: "subcription/unlock",
					Name:      "Unlock",
					Meta: Meta{
						Title:     "unlocktest",
						Icon:      "security",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
				{
					Path:      "tcp",
					Component: "test/tcp",
					Name:      "TcpTest",
					Meta: Meta{
						Title:     "tcptest",
						Icon:      "link",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
			},
		},
		// 模板管理
		{
			Path:      "/template",
			Component: "Layout",
			Redirect:  "/template/list",
			Name:      "template",
			Meta: Meta{
				Title:  "template",
				Icon:   "document",
				Hidden: false,
				Roles:  []string{"ADMIN"},
			},
			Children: []Child{
				{
					Path:      "list",
					Component: "template/list",
					Name:      "TemplateList",
					Meta: Meta{
						Title:     "templatelist",
						Icon:      "document",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
				{
					Path:      "builder",
					Component: "template/builder",
					Name:      "TemplateBuilder",
					Meta: Meta{
						Title:     "templatebuilder",
						Icon:      "edit",
						Hidden:    false,
						Roles:     []string{"ADMIN"},
						KeepAlive: true,
					},
				},
			},
		},
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": menus,
		"msg":  "获取成功",
	})
}
