package model

type LoginLog struct {
	ID        uint     `gorm:"primaryKey" json:"id"`
	Username  string   `gorm:"size:64" json:"username"`
	Status    int      `json:"status"`
	Message   string   `gorm:"size:255" json:"message"`
	IP        string   `gorm:"size:45" json:"ip"`
	CreatedAt DateTime `json:"created_at"`
}

func (LoginLog) TableName() string { return "login_logs" }
