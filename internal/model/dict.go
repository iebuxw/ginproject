package model

import "time"

// DictType 字典类型
type DictType struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Code        string    `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Description string    `gorm:"size:255" json:"description"`
	Status      int       `gorm:"default:1" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DictType) TableName() string { return "dict_types" }

// DictData 字典数据
type DictData struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DictTypeID uint      `gorm:"index;not null" json:"dict_type_id"`
	Label      string    `gorm:"size:64;not null" json:"label"`
	Value      string    `gorm:"size:64;not null" json:"value"`
	Sort       int       `gorm:"default:0" json:"sort"`
	Status     int       `gorm:"default:1" json:"status"`
	Remark     string    `gorm:"size:255" json:"remark"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (DictData) TableName() string { return "dict_data" }
