package models

import (
	"time"
	"gorm.io/gorm"
)

type Airport struct {
	gorm.Model
	ID          int
	Name        string     `gorm:"uniqueIndex"`
	URL         string
	AutoCleanup bool       // 是否自动清理死节点
	IsDedicated bool       // 是否为专线（测活失败时不丢弃）
	LastSync       *time.Time // 上次同步时间
	NodeCount      int        // 节点总数
	SelectedNodes  string     `gorm:"type:text"` // 逗号分隔的已勾选节点名，空=全选
}

func (a *Airport) Add() error {
	return DB.Create(a).Error
}

func (a *Airport) Update() error {
	return DB.Save(a).Error
}

func (a *Airport) Delete() error {
	return DB.Delete(a).Error
}

func GetAirports() ([]Airport, error) {
	var airports []Airport
	err := DB.Find(&airports).Error
	return airports, err
}

func (a *Airport) Find() error {
	return DB.First(a, a.ID).Error
}
