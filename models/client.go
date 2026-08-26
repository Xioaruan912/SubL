package models

import "gorm.io/gorm"

// ClientVersion 客户端下载记录（每客户端每平台一条）
type ClientVersion struct {
	gorm.Model
	Client    string // 客户端名：clash-verge-rev / v2rayN / FlClash
	Platform  string // 平台：win-x64 / win-arm64 / mac-x64 / mac-arm64
	Version   string // 已下载版本 tag（如 v2.5.2 / 7.24.4）
	FileName  string // 磁盘文件名
	Size      int64  // 字节
	Status    string // idle / downloading / ready / failed
	ErrMsg    string
	UpdatedAt int64 // Unix 秒
}

// List 按客户端/平台查全部记录
func (cv *ClientVersion) List() ([]ClientVersion, error) {
	var list []ClientVersion
	err := DB.Find(&list).Error
	return list, err
}

// ByClientPlatform 查询单条记录
func (cv *ClientVersion) ByClientPlatform(client, platform string) (*ClientVersion, error) {
	var rec ClientVersion
	err := DB.Where("client = ? AND platform = ?", client, platform).First(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Save 新增或更新（按 client+platform）
func (cv *ClientVersion) Save() error {
	var rec ClientVersion
	err := DB.Where("client = ? AND platform = ?", cv.Client, cv.Platform).First(&rec).Error
	if err == gorm.ErrRecordNotFound {
		cv.UpdatedAt = cv.UpdatedAt
		return DB.Create(cv).Error
	}
	if err != nil {
		return err
	}
	cv.ID = rec.ID
	cv.CreatedAt = rec.CreatedAt
	return DB.Model(&rec).Updates(map[string]any{
		"version":    cv.Version,
		"file_name":  cv.FileName,
		"size":       cv.Size,
		"status":     cv.Status,
		"err_msg":    cv.ErrMsg,
		"updated_at": cv.UpdatedAt,
	}).Error
}

// SetStatus 更新状态
func (cv *ClientVersion) SetStatus(status, errMsg string) error {
	return DB.Model(&ClientVersion{}).Where("client = ? AND platform = ?", cv.Client, cv.Platform).Updates(map[string]any{
		"status": status,
		"err_msg": errMsg,
	}).Error
}