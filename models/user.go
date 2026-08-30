package models

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int
	Username string
	Password string
	Role     string
	Nickname string
}

func passwordIsHashed(value string) bool {
	return strings.HasPrefix(value, "$2a$") || strings.HasPrefix(value, "$2b$") || strings.HasPrefix(value, "$2y$")
}

func HashPassword(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("password empty")
	}
	if passwordIsHashed(value) {
		return value, nil
	}
	out, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.DefaultCost)
	return string(out), err
}

func (user *User) Create() error { // 创建用户
	hashed, err := HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashed
	return DB.Create(user).Error
}
func (user *User) Set(UpdateUser *User) error { // 设置用户
	if UpdateUser.Password != "" {
		hashed, err := HashPassword(UpdateUser.Password)
		if err != nil {
			return err
		}
		UpdateUser.Password = hashed
	}
	return DB.Where("username = ?", user.Username).Updates(UpdateUser).Error
}
func (user *User) Verify() error { // 验证用户
	plain := user.Password
	var stored User
	if err := DB.Where("username = ?", user.Username).First(&stored).Error; err != nil {
		return err
	}
	if passwordIsHashed(stored.Password) {
		if err := bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte(plain)); err != nil {
			return err
		}
	} else {
		if stored.Password != plain {
			return fmt.Errorf("password mismatch")
		}
		hashed, err := HashPassword(plain)
		if err == nil {
			_ = DB.Model(&stored).Update("password", hashed).Error
			stored.Password = hashed
		}
	}
	*user = stored
	return nil
}

func (user *User) Find() error { // 查找用户
	return DB.Where("username = ? ", user.Username).First(user).Error
}

func (user *User) All() ([]User, error) { // 获取所有用户
	var users []User
	err := DB.Find(&users).Error
	return users, err
}

func (user *User) Del() error { // 删除用户
	return DB.Delete(user).Error
}
