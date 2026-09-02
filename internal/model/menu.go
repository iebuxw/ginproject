package model

import "time"

type Menu struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ParentID   uint      `gorm:"default:0" json:"parent_id"`
	Name       string    `gorm:"size:64;not null" json:"name"`
	Icon       string    `gorm:"size:64" json:"icon"`
	Path       string    `gorm:"size:255" json:"path"`
	Type       int       `gorm:"not null" json:"type"`
	Permission string    `gorm:"size:128" json:"permission"`
	Sort       int       `gorm:"default:0" json:"sort"`
	Status     int       `gorm:"default:1" json:"status"`
	Children   []Menu    `gorm:"-" json:"children,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MenuItemSort 批量排序请求项
type MenuItemSort struct {
	ID   uint `json:"id" binding:"required"`
	Sort int  `json:"sort"`
}

func (Menu) TableName() string { return "menus" }
