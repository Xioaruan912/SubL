package models

import "time"

// RoutingRegressionCase stores an explicit expected routing outcome. Unlike an
// egress target, it may intentionally pin an expected policy/country because it
// is a regression assertion, not a generic probe definition.
type RoutingRegressionCase struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:120;not null" json:"name"`
	Domain          string    `gorm:"size:255;index;not null" json:"domain"`
	ExpectedPolicy  string    `gorm:"size:255" json:"expectedPolicy"`
	ExpectedCountry string    `gorm:"size:8" json:"expectedCountry"`
	ForbiddenPolicy string    `gorm:"size:255" json:"forbiddenPolicy"`
	Protocol        string    `gorm:"size:16" json:"protocol"`
	Port            int       `json:"port"`
	Enabled         bool      `gorm:"not null;default:true;index" json:"enabled"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
