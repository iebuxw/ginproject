package model

type DbBackup struct {
	ID          int64    `json:"id" gorm:"primaryKey;autoIncrement"`
	Filename    string   `json:"filename" gorm:"size:255;not null"`
	FileSize    int64    `json:"file_size"`
	TriggerType string   `json:"trigger_type" gorm:"size:20"`
	Status      int      `json:"status"`
	Type        string   `json:"type" gorm:"size:20"`
	Remark      string   `json:"remark" gorm:"type:text"`
	CreatedAt   DateTime `json:"created_at"`
}

func (DbBackup) TableName() string {
	return "db_backups"
}
