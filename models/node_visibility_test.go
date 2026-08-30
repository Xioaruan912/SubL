package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNodeGroupingAndGlobalVisibility(t *testing.T) {
	oldDB := DB
	t.Cleanup(func() { DB = oldDB })

	db, err := gorm.Open(sqlite.Open("file:node-visibility?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	DB = db
	if err := db.AutoMigrate(&Node{}, &GroupNode{}); err != nil {
		t.Fatal(err)
	}

	node := Node{Name: "JP-01", Link: "ss://example"}
	if err := node.Add(); err != nil {
		t.Fatal(err)
	}
	var loaded Node
	if err := db.Preload("GroupNodes").First(&loaded, node.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(loaded.GroupNodes) != 1 || loaded.GroupNodes[0].Name != DefaultNodeGroupName {
		t.Fatalf("new node must have default group, got %#v", loaded.GroupNodes)
	}

	group := GroupNode{Name: "机场 A"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := loaded.UpdateGroup([]GroupNode{group}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&group).Update("hidden", true).Error; err != nil {
		t.Fatal(err)
	}
	visible, err := GetNodeList()
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 0 {
		t.Fatalf("node in hidden group must be globally hidden, got %d", len(visible))
	}

	if err := db.Model(&group).Update("hidden", false).Error; err != nil {
		t.Fatal(err)
	}
	visible, err = GetNodeList()
	if err != nil || len(visible) != 1 {
		t.Fatalf("unhidden group should restore node, len=%d err=%v", len(visible), err)
	}
	if err := db.Model(&loaded).Update("hidden", true).Error; err != nil {
		t.Fatal(err)
	}
	visible, err = GetNodeList()
	if err != nil || len(visible) != 0 {
		t.Fatalf("directly hidden node must stay hidden, len=%d err=%v", len(visible), err)
	}
	all, err := GetAllNodeList()
	if err != nil || len(all) != 1 {
		t.Fatalf("hidden data must remain recoverable, len=%d err=%v", len(all), err)
	}
}
