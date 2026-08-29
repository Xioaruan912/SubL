package models

import "time"

// TemplateVersion keeps an immutable snapshot before and after template edits.
// Templates remain plain files, so existing rendering logic is untouched.
type TemplateVersion struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Filename  string    `gorm:"index;size:255;not null" json:"filename"`
	Content   string    `gorm:"type:text;not null" json:"content,omitempty"`
	Action    string    `gorm:"size:32;not null" json:"action"`
	CreatedAt time.Time `gorm:"index;not null" json:"createdAt"`
}

func SaveTemplateVersion(filename, content, action string) error {
	version := TemplateVersion{Filename: filename, Content: content, Action: action, CreatedAt: time.Now()}
	if err := DB.Create(&version).Error; err != nil {
		return err
	}
	// Keep the newest 50 versions per template.
	return DB.Exec(`DELETE FROM template_versions WHERE filename = ? AND id NOT IN
		(SELECT id FROM template_versions WHERE filename = ? ORDER BY id DESC LIMIT 50)`, filename, filename).Error
}
