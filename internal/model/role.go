package model

import "time"

type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Code        string    `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Description string    `gorm:"size:255" json:"description"`
	Status      int       `gorm:"default:1" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// many2many:role_menus：Role 和 Menu 是【多对多】关系，用一张中间表来保存关联
	// omitempty：如果 Menus 是 nil / 空切片，返回 json 的时候就直接把这个字段删掉，不返回
	Menus       []Menu    `gorm:"many2many:role_menus;" json:"menus,omitempty"`
}

func (Role) TableName() string { return "roles" }
