package models

import (
	"errors"
	"log"

	"gorm.io/gorm"
)

const DefaultNodeGroupName = "默认"

type GroupNode struct {
	gorm.Model
	ID        int
	Name      string
	Hidden    bool   `gorm:"not null;default:false;index" json:"Hidden"`
	Nodes     []Node `gorm:"many2many:group_node_nodes"` // 多对多关联字段
	NodeCount int    `gorm:"-"`                          // 仅用于前端展示，不入库
}

type Node struct {
	gorm.Model
	ID         int
	Name       string
	Link       string
	Hidden     bool        `gorm:"not null;default:false;index" json:"Hidden"`
	GroupNodes []GroupNode `gorm:"many2many:group_node_nodes"` // 反向关联字段
}

// hook Node 写入创建删除修改 等写入权限
func (n *Node) AfterSave(*gorm.DB) error {
	// 写操作前执行（Create 或 Update）

	return nil
}

// Find 根据 ID 查找节点
func (n *Node) Find() error {
	return DB.Where("id = ?", n.ID).First(n).Error
}

// 创建分组
func (gn *GroupNode) Add() error {
	// 检查分组是否已存在
	var existingGroup GroupNode
	result := DB.Model(gn).Where("name = ?", gn.Name).First(&existingGroup) // 查询数据库中是否存在同名的分组
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Println(result.Error)
		return result.Error // 如果查询出错，返回错误
	}
	if result.RowsAffected > 0 { // 如果查询到分组已存在
		// log.Println("分组已存在")
		return nil // 不返回错误存在就跳过

	}
	return DB.FirstOrCreate(gn, GroupNode{Name: gn.Name}).Error
}

// 关联分组
func (gn *GroupNode) Ass(n *Node) error {
	result := DB.Model(gn).Where("name = ?", gn.Name).First(gn) // 查找分组
	// log.Println("分组ID:", gn.ID, "分组昵称:", gn.Name, "错误信息:", result.Error)
	if result.Error != nil {
		log.Println(result.Error)
	}
	result = DB.Model(n).Where("name = ?", n.Name).First(n) // 查找节点
	// log.Println("节点ID:", n.ID, "节点昵称:", n.Name, "错误信息:", result.Error)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return DB.Model(&gn).Association("Nodes").Append(n)
}

// UnbindGroup 将节点从指定分组解除绑定（不删除节点）
func (n *Node) UnbindGroup(groupName string) error {
	var gn GroupNode
	if err := DB.Where("name = ?", groupName).First(&gn).Error; err != nil {
		return err
	}
	if err := DB.Where("name = ?", n.Name).First(n).Error; err != nil {
		return err
	}
	return DB.Model(&gn).Association("Nodes").Delete(n)
}

// 更新分组信息
func (gn *GroupNode) Update(NewGn *GroupNode) error {
	// 读取分组数据
	var FirstGn GroupNode
	result := DB.Model(gn).Where("id = ? or name = ?", NewGn.ID, NewGn.Name).First(&FirstGn)
	if result.Error != nil {
		log.Println(result.Error)
		return result.Error
	}
	if result.RowsAffected > 0 {
		return errors.New("分组已存在")
	}
	return DB.Model(gn).Where("id = ? or name = ?", gn.ID, gn.Name).Updates(&NewGn).Error
}

// 删除分组
func (gn *GroupNode) Del() error {
	// 读取分组数据
	result := DB.Model(gn).Where("id = ? or name = ?", gn.ID, gn.Name).First(&gn)
	if result.Error != nil {
		log.Println(result.Error)
		return result.Error
	}
	// 读取分组关联的节点数据
	result = DB.Model(gn).Preload("Nodes").First(gn) // 预加载分组关联的节点数据
	if result.Error != nil {
		log.Println(result.Error)
		return result.Error
	}
	log.Println(gn.Nodes)
	err := DB.Model(gn).Association("Nodes").Delete(gn.Nodes)
	if err != nil {
		log.Println("解除关联失败", err)
		return err
	}
	return DB.Model(gn).Where("id = ? or name = ?", gn.ID, gn.Name).Delete(gn).Error // 删除分组记录
}

// 查看所有分组

func GetGroupNodeList() ([]GroupNode, error) {
	var gns []GroupNode
	result := DB.Model(gns).Where("hidden = ?", false).Preload("Nodes", "hidden = ?", false).Find(&gns)
	if result.Error != nil {
		return nil, errors.New("没有任何分组")
	}
	hiddenIDs, _ := GloballyHiddenNodeIDs()
	for i := range gns {
		visible := gns[i].Nodes[:0]
		for _, n := range gns[i].Nodes {
			if !hiddenIDs[n.ID] {
				visible = append(visible, n)
			}
		}
		gns[i].Nodes = visible
	}
	return gns, result.Error
}

func GetAllGroupNodeList() ([]GroupNode, error) {
	var gns []GroupNode
	err := DB.Model(&GroupNode{}).Preload("Nodes").Order("id asc").Find(&gns).Error
	return gns, err
}

/* 下面为节点的增删改查 */

// 添加节点的方法
func (n *Node) Add() error {
	// 检查节点是否已存在
	var existingNode Node
	result := DB.Model(n).Where("link = ? and name =?", n.Link, n.Name).First(&existingNode) // 查询数据库中是否存在同名同链接的节点
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error // 如果查询出错，返回错误
	}
	if result.RowsAffected > 0 {
		*n = existingNode
		return EnsureNodeHasGroup(n.ID)
	}
	if err := DB.Model(n).Create(n).Error; err != nil {
		return err
	}
	return EnsureNodeHasGroup(n.ID)
}

// 删除节点
func (n *Node) Del() error {
	// 查看是否有关联 有的话解除关联
	DB.Model(n).Preload("GroupNodes").First(n) // 预加载分组节点
	gns := n.GroupNodes

	if len(n.GroupNodes) > 0 {
		err := DB.Model(n).Association("GroupNodes").Delete(n.GroupNodes)
		if err != nil {
			return err
		}
	}
	IsGroupNotDel(gns)
	// 如果分组节点没有关联的节点则删除分组节点
	// for _, gn := range gns {
	// 	DB.Model(gn).Preload("Nodes").Find(&gn) // 预加载分组节点数据
	// 	log.Println("gnNodes:", gn.Nodes)
	// 	if len(gn.Nodes) == 0 {
	// 		// log.Println("分组节点没有关联的节点，删除分组节点", gn.Name)
	// 		err := DB.Model(gn).Delete(&gn).Error // 删除分组节点
	// 		if err != nil {
	// 			log.Println("删除分组节点失败", err)
	// 			return err
	// 		}
	// 	}
	// }
	// Unscoped  硬删除
	// 默认删除是软删除 数据库仍然存在记录
	return DB.Model(n).Delete(n).Error
}

// 更新节点

func (n *Node) UpdateNode(New *Node) error {
	// 检查节点是否已存在
	var n1 Node
	result := DB.Model(n).Where("id = ?", New.ID).First(&n1) // 查询数据库中是否存在同名同链接的节点
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Println(result.Error)
		return result.Error // 如果查询出错，返回错误

	}
	if result.RowsAffected > 0 {
		log.Println("节点已经存在", result.Error)
		return errors.New("节点已经存在") // 如果查询出错，返回错误
	}
	// 更新记录
	return DB.Model(n).Where("id = ?", n.ID).Updates(New).Error
}

// 检查分组无绑定则删除
func IsGroupNotDel(gns []GroupNode) error {
	// 如果分组节点没有关联的节点则删除分组节点
	for _, gn := range gns {
		DB.Model(gn).Preload("Nodes").Find(&gn) // 预加载分组节点数据
		// log.Println("gnNodes:", gn.Nodes, "长度:", len(gn.Nodes))
		if len(gn.Nodes) == 0 {
			// log.Println("分组节点没有关联的节点，删除分组节点", gn.Name)
			err := DB.Model(gn).Delete(&gn).Error // 删除分组节点
			if err != nil {
				log.Println("删除分组节点失败", err)
				return err
			}
		}
	}
	return nil
}

// 更新关联分组

func (n *Node) UpdateGroup(gns []GroupNode) error {
	if len(gns) == 0 {
		return errors.New("节点必须至少保留一个分组")
	}
	// 检测节点是否存在
	result := DB.Model(n).Where("id = ? or name = ?", n.ID, n.Name).First(&n) // 查找节点
	if result.Error != nil {
		log.Println(result.Error)
		return result.Error // 如果查询出错，返回错误
	}

	// 检查分组是否已存在
	var NewGroupDatas []GroupNode

	for _, gn := range gns {

		// var NewGroupData GroupNode

		if gn.Name == "" {

			// 预加载关联
			result := DB.Model(n).Preload("GroupNodes").First(n) // 预加载分组节点
			if result.Error != nil {
				log.Println(result.Error)
				return result.Error // 如果查询出错，返回错误
			}
			IsGroupNot := n.GroupNodes // 临时分组节点切片
			log.Println("NewGroupDatas", IsGroupNot)

			// 解除关联
			// log.Println("分组名称为空,解除关联", NewGroup)
			err := DB.Model(n).Association("GroupNodes").Clear()
			if err != nil {
				log.Println("解除关联失败", err)
				return err
			}
			//

			err = IsGroupNotDel(IsGroupNot) // 检查分组节点是否有绑定的节点，如果没有则删除分组节点
			if err != nil {
				log.Println(err)
				// return err // 如果检查分组节点失败，返回错误
			}
			return nil
		}
		result := DB.Model(gn).Where("name = ?", gn.Name).First(&gn) // 查找分组
		// 没有找到记录
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Println(result.Error)
			return result.Error // 如果查询出错，返回错误
		}
		NewGroupDatas = append(NewGroupDatas, gn) // 将新的数据添加到更新列表中
	}

	// 更新记录
	return DB.Model(n).Association("GroupNodes").Replace(NewGroupDatas) // 替换分组节点
}

// 查看所有节点

func GetNodeList() ([]Node, error) {
	hiddenIDs, err := GloballyHiddenNodeIDs()
	if err != nil {
		return nil, err
	}
	var ns []Node
	result := DB.Model(&Node{}).Where("hidden = ?", false).Preload("GroupNodes", "hidden = ?", false).Find(&ns)
	if result.Error != nil {
		return nil, result.Error
	}
	visible := ns[:0]
	for _, n := range ns {
		if !hiddenIDs[n.ID] {
			visible = append(visible, n)
		}
	}
	return visible, nil
}

func GetAllNodeList() ([]Node, error) {
	var ns []Node
	result := DB.Model(&Node{}).Preload("GroupNodes").Order("id asc").Find(&ns)
	if result.Error != nil {
		return nil, result.Error
	}
	return ns, result.Error
}

// GloballyHiddenNodeIDs returns nodes hidden directly or indirectly by a hidden group.
func GloballyHiddenNodeIDs() (map[int]bool, error) {
	result := map[int]bool{}
	var direct []int
	if err := DB.Model(&Node{}).Where("hidden = ?", true).Pluck("id", &direct).Error; err != nil {
		return nil, err
	}
	for _, id := range direct {
		result[id] = true
	}
	var grouped []int
	err := DB.Table("group_node_nodes AS x").Distinct("x.node_id").
		Joins("JOIN group_nodes AS g ON g.id = x.group_node_id AND g.deleted_at IS NULL").
		Where("g.hidden = ?", true).Pluck("x.node_id", &grouped).Error
	if err != nil {
		return nil, err
	}
	for _, id := range grouped {
		result[id] = true
	}
	return result, nil
}

func FilterVisibleNodes(nodes []Node) []Node {
	hiddenIDs, err := GloballyHiddenNodeIDs()
	if err != nil {
		return nodes
	}
	result := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		if !n.Hidden && !hiddenIDs[n.ID] {
			result = append(result, n)
		}
	}
	return result
}

func EnsureNodeHasGroup(nodeID int) error {
	if nodeID <= 0 {
		return nil
	}
	var count int64
	if err := DB.Table("group_node_nodes AS x").
		Joins("JOIN group_nodes AS g ON g.id = x.group_node_id AND g.deleted_at IS NULL").
		Where("x.node_id = ?", nodeID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	var group GroupNode
	if err := DB.Where("name = ?", DefaultNodeGroupName).FirstOrCreate(&group, GroupNode{Name: DefaultNodeGroupName}).Error; err != nil {
		return err
	}
	var node Node
	if err := DB.First(&node, nodeID).Error; err != nil {
		return err
	}
	return DB.Model(&node).Association("GroupNodes").Append(&group)
}

func EnsureNodeGroupMembership() error {
	var ids []int
	err := DB.Model(&Node{}).
		Where("NOT EXISTS (SELECT 1 FROM group_node_nodes x JOIN group_nodes g ON g.id = x.group_node_id AND g.deleted_at IS NULL WHERE x.node_id = nodes.id)").
		Pluck("id", &ids).Error
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := EnsureNodeHasGroup(id); err != nil {
			return err
		}
	}
	return nil
}
