package models

import (
	"time"

	"gorm.io/gorm"
)

// Collection 多订阅合集：把若干订阅的节点合并为一个统一下发链接。
type Collection struct {
	gorm.Model
	ID        int
	Name      string     `gorm:"size:160;not null" json:"name"`
	Token     string     `gorm:"size:64;index" json:"token"`
	MemberIDs string     `gorm:"type:text" json:"memberIds"` // JSON 数组，订阅 ID
	Dedupe    bool       `gorm:"not null;default:true" json:"dedupe"`
	Pipeline  string     `gorm:"type:text" json:"pipeline"`
	Config    string     `gorm:"type:text" json:"config"`
	Note      string     `gorm:"size:255" json:"note"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// EnsureToken 确保合集有令牌。
func (col *Collection) EnsureToken() error {
	if col.ID == 0 || col.Token != "" {
		return nil
	}
	col.Token = GenerateToken()
	return DB.Model(col).Update("token", col.Token).Error
}
