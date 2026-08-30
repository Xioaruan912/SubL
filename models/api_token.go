package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type APIToken struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"size:120;not null" json:"name"`
	TokenPrefix string    `gorm:"size:16;index" json:"tokenPrefix"`
	TokenHash  string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	Scopes     string     `gorm:"size:120;not null" json:"scopes"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	Enabled    bool       `gorm:"not null;default:true;index" json:"enabled"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func GenerateAPIToken() (string, string, string, error) {
	b:=make([]byte,32);if _,err:=rand.Read(b);err!=nil{return "","","",err};plain:="slx_"+hex.EncodeToString(b);sum:=sha256.Sum256([]byte(plain));prefix:=plain;if len(prefix)>12{prefix=prefix[:12]};return plain,prefix,hex.EncodeToString(sum[:]),nil
}

func HashAPIToken(value string) string { sum:=sha256.Sum256([]byte(strings.TrimSpace(value)));return hex.EncodeToString(sum[:]) }

func (t APIToken) HasScope(scope string) bool {
	want:=strings.ToLower(strings.TrimSpace(scope));for _,item:=range strings.Split(strings.ToLower(t.Scopes),","){item=strings.TrimSpace(item);if item=="admin"||item==want{return true};if want=="read"&&item=="write"{return true}};return false
}
