package model

import "time"

type OperationLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	OperatorID uint      `json:"operator_id"`
	Module     string    `gorm:"size:64" json:"module"`
	Action     string    `gorm:"size:64" json:"action"`
	Method     string    `gorm:"size:10" json:"method"`
	Path       string    `gorm:"size:255" json:"path"`
	Params     string    `gorm:"type:text" json:"params"`
	Response   string    `gorm:"type:text" json:"response"`
	Duration   int       `json:"duration"`
	IP         string    `gorm:"size:45" json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}

func (OperationLog) TableName() string { return "operation_logs" }
