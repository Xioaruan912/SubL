package models

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestPasswordHashAndLegacyMigration(t *testing.T) {
	hashed,err:=HashPassword("secret-123");if err!=nil{t.Fatal(err)};if hashed=="secret-123"||!strings.HasPrefix(hashed,"$2"){t.Fatalf("password was not bcrypt hashed: %q",hashed)};if err:=bcrypt.CompareHashAndPassword([]byte(hashed),[]byte("secret-123"));err!=nil{t.Fatalf("bcrypt verify failed: %v",err)}
	old:=DB;db,err:=gorm.Open(sqlite.Open(":memory:"),&gorm.Config{});if err!=nil{t.Fatal(err)};DB=db;t.Cleanup(func(){DB=old});if err:=db.AutoMigrate(&User{});err!=nil{t.Fatal(err)};if err:=db.Exec("INSERT INTO users (username,password,role,nickname) VALUES (?,?,?,?)","legacy","plain-pass","admin","Legacy").Error;err!=nil{t.Fatal(err)}
	user:=&User{Username:"legacy",Password:"plain-pass"};if err:=user.Verify();err!=nil{t.Fatalf("legacy login failed: %v",err)};var stored User;if err:=db.Where("username = ?","legacy").First(&stored).Error;err!=nil{t.Fatal(err)};if stored.Password=="plain-pass"||!strings.HasPrefix(stored.Password,"$2"){t.Fatalf("legacy password not migrated: %q",stored.Password)}
}

func TestAPITokenScopes(t *testing.T){read:=APIToken{Scopes:"read"};if !read.HasScope("read")||read.HasScope("write"){t.Fatal("read scope permissions incorrect")};write:=APIToken{Scopes:"write"};if !write.HasScope("read")||!write.HasScope("write"){t.Fatal("write should imply read")};admin:=APIToken{Scopes:"admin"};if !admin.HasScope("read")||!admin.HasScope("write")||!admin.HasScope("admin"){t.Fatal("admin should imply all scopes")}}
