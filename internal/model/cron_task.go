package model

import "time"

// CronTask 定时任务
type CronTask struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"size:64;not null" json:"name"`
	URL            string    `gorm:"size:255;not null" json:"url"`
	Method         string    `gorm:"size:8;not null;default:GET" json:"method"`
	Headers        string    `gorm:"type:text" json:"headers"`
	Body           string    `gorm:"type:text" json:"body"`
	Cron           string    `gorm:"size:32;not null" json:"cron"`
	Timeout        int       `gorm:"not null;default:30" json:"timeout"`
	Status         int       `gorm:"not null;default:1" json:"status"`
	Remark         string    `gorm:"size:255;default:''" json:"remark"`
	LastExecStatus int       `gorm:"->" json:"last_exec_status"` // 最近一次执行状态，-1=无记录（只读，列表子查询填充）
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (CronTask) TableName() string { return "cron_tasks" }

// CronTaskExecution 定时任务执行日志
type CronTaskExecution struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TaskID     uint      `gorm:"index;not null" json:"task_id"`
	Trigger    string    `gorm:"size:16;not null" json:"trigger"`
	Status     int       `gorm:"not null" json:"status"`
	HTTPStatus int       `json:"http_status"`
	Response   string    `gorm:"type:text" json:"response"`
	ErrorMsg   string    `gorm:"size:255;default:''" json:"error_msg"`
	DurationMS int       `json:"duration_ms"`
	CreatedAt  time.Time `json:"created_at"`
}

func (CronTaskExecution) TableName() string { return "cron_task_executions" }
