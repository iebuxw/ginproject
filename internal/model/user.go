package model

import "time"

type User struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Username    string     `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password    string     `gorm:"size:255;not null" json:"-"`
	Email       string     `gorm:"size:128" json:"email"`
	Phone       string     `gorm:"size:20" json:"phone"`
	Status      int        `gorm:"default:1" json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Roles       []Role     `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

func (User) TableName() string { return "users" }
