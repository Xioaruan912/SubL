package models

import "time"

// ShortLink 内部短链：短码 -> 目标地址（通常是订阅地址或面板地址）。
type ShortLink struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Target    string    `gorm:"type:text;not null" json:"target"`
	Remark    string    `gorm:"size:160" json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
}
