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
	Menus       []Menu    `gorm:"many2many:role_menus;" json:"menus,omitempty"`
}

func (Role) TableName() string { return "roles" }
