package models

import (
	// 用于将配置解析为结构体
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings" // 用于处理逗号分隔的字符串
	"time"

	"gorm.io/gorm"
)

// Subcription 结构体
type Subcription struct {
	gorm.Model
	ID        int
	Name      string
	Config    string     `gorm:"type:text"` // Config 存储为 JSON 字符串
	NodeOrder string     `gorm:"type:text"`
	Token     string     `gorm:"type:text"` // 订阅链接身份令牌（随机生成，可重置）
	ExpiresAt *time.Time  // 过期时间，nil 表示永不过期
	Nodes     []Node      `gorm:"many2many:subcription_nodes;"`
	GroupRefs []GroupNode `gorm:"many2many:subcription_groups;"` // 引用的分组（机场同步后节点自动跟进）
	SubLogs   []SubLogs   `gorm:"foreignKey:SubcriptionID;"`
}

// Config 结构体，用于解析 Subcription.Config 字段的 JSON 内容
// 命名为 SubscriptionConfig 以避免与其他可能的 Config 冲突
type SubscriptionConfig struct { // <--- 这里重命名了
	Clash string `json:"clash"`
	Surge string `json:"surge"`
	Loon  string `json:"loon"`
	UDP   bool   `json:"udp"`
	Cert  bool   `json:"cert"`
}

// GenerateToken 生成随机订阅令牌（32 位十六进制）
func GenerateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 极罕见失败，退回时间戳哈希
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b)
}

// EnsureToken 确保订阅有令牌，没有则生成并保存
func (sub *Subcription) EnsureToken() error {
	if sub.ID == 0 {
		return nil
	}
	if sub.Token != "" {
		return nil
	}
	sub.Token = GenerateToken()
	return DB.Model(sub).Update("token", sub.Token).Error
}

// Add 添加订阅
func (sub *Subcription) Add() error {
	// 在创建订阅时，如果 sub.Nodes 已经被前端填充并排序，可以将其名称转换为 NodeOrder 字符串
	if len(sub.Nodes) > 0 {
		names := make([]string, len(sub.Nodes))
		for i, node := range sub.Nodes {
			names[i] = node.Name
		}
		sub.NodeOrder = strings.Join(names, ",")
	}
	// 生成订阅令牌
	if sub.Token == "" {
		sub.Token = GenerateToken()
	}

	// 首先创建 Subcription 记录，不包括多对多关系
	if err := DB.Create(sub).Error; err != nil {
		return err
	}
	// 然后建立多对多关系

	// log.Println("Adding subscription nodes:", sub.Nodes)
	if err := DB.Model(sub).Association("Nodes").Append(sub.Nodes); err != nil {
		return err
	}
	// 建立分组引用关系
	return DB.Model(sub).Association("GroupRefs").Replace(sub.GroupRefs)
}

// Update 更新订阅
func (sub *Subcription) Update(NewName *Subcription) error {
	// 查找现有订阅
	var existingSub Subcription
	if err := DB.Where("id = ? or name = ?", sub.ID, sub.Name).First(&existingSub).Error; err != nil {
		return err // 订阅不存在
	}

	// 更新非多对多字段，包括 NodeOrder
	existingSub.Name = NewName.Name // 新名称
	existingSub.Config = NewName.Config

	// 更新 NodeOrder 字段
	if len(NewName.Nodes) > 0 {
		names := make([]string, len(NewName.Nodes))
		for i, node := range NewName.Nodes {
			names[i] = node.Name
		}
		existingSub.NodeOrder = strings.Join(names, ",")
	} else {
		existingSub.NodeOrder = "" // 如果没有节点，清空
	}

	// 保存更新
	if err := DB.Save(&existingSub).Error; err != nil {
		return err
	}

	// 更新多对多关系: Replace 会清除旧关联并建立新关联
	// 确保 sub.Nodes 包含了新的排序后的节点对象
	log.Println("Updating subscription nodes:", NewName.SubLogs)
	if err := DB.Model(&existingSub).Association("Nodes").Replace(NewName.Nodes); err != nil {
		return err
	}
	// 更新分组引用关系
	return DB.Model(&existingSub).Association("GroupRefs").Replace(NewName.GroupRefs)
}

// Find 查找订阅 (通常用于获取单个订阅的详细信息，包括其关联节点和日志)
func (sub *Subcription) Find() error {
	// 使用 Preload 加载 Nodes 和 SubLogs 关联数据
	if err := DB.Preload("Nodes").Preload("SubLogs").Where("id = ? or name = ?", sub.ID, sub.Name).First(sub).Error; err != nil {
		return err
	}
	// 根据 NodeOrder 字段重新排序 Nodes
	if sub.NodeOrder != "" && len(sub.Nodes) > 0 {
		orderedNames := strings.Split(sub.NodeOrder, ",")
		nodeMap := make(map[string]Node)
		for _, node := range sub.Nodes {
			log.Println("node:", node)
			nodeMap[node.Name] = node
		}

		var reorderedNodes []Node
		for _, name := range orderedNames {
			trimmedName := strings.TrimSpace(name)
			if node, ok := nodeMap[trimmedName]; ok {
				reorderedNodes = append(reorderedNodes, node)
			}
		}
		sub.Nodes = reorderedNodes
	}

	return nil
}

// List 订阅列表 (返回所有订阅，并加载其关联节点和日志，按指定顺序)
func (sub *Subcription) List() ([]Subcription, error) {
	var subs []Subcription
	err := DB.Preload("Nodes").Preload("SubLogs").Find(&subs).Error // 预加载所有关联
	if err != nil {
		return nil, err
	}

	for i := range subs {
		// 确保老数据有令牌（惰性生成并落库）
		_ = subs[i].EnsureToken()
		// 根据 NodeOrder 字段重新排序每个订阅的 Nodes
		if subs[i].NodeOrder != "" && len(subs[i].Nodes) > 0 {
			orderedNames := strings.Split(subs[i].NodeOrder, ",")
			nodeMap := make(map[string]Node) // 用于快速查找节点对象
			for _, node := range subs[i].Nodes {
				nodeMap[node.Name] = node
			}

			var reorderedNodes []Node
			for _, name := range orderedNames {
				trimmedName := strings.TrimSpace(name)
				if node, ok := nodeMap[trimmedName]; ok {
					reorderedNodes = append(reorderedNodes, node)
				}
			}
			subs[i].Nodes = reorderedNodes
		}
	}
	return subs, nil
}

// IPlogUpdate 更新订阅日志 (与节点排序无关，保持不变)
func (sub *Subcription) IPlogUpdate() error {
	return DB.Model(sub).Association("SubLogs").Replace(&sub.SubLogs)
}

// Del 删除订阅 (与节点排序无关，保持不变)
func (sub *Subcription) Del() error {
	// 清除多对多关系
	err := DB.Model(sub).Association("Nodes").Clear()
	if err != nil {
		return err
	}
	// 清除分组引用关系
	err = DB.Model(sub).Association("GroupRefs").Clear()
	if err != nil {
		return err
	}
	// 删除主记录，由于 SubLogs 使用 foreignKey，理论上 GORM 应该能级联删除子记录。
	// 但为了确保，你也可以显式删除 SubLogs:
	// DB.Where("subcription_id = ?", sub.ID).Delete(&SubLogs{})
	return DB.Delete(sub).Error
}
