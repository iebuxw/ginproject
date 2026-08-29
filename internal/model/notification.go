package model

// Notification 消息通知本体
type Notification struct {
	ID         uint     `gorm:"primaryKey" json:"id"`
	Type       int      `json:"type"` // 1=公告 2=站内信 3=系统事件
	Title      string   `gorm:"size:200;not null" json:"title"`
	Content    string   `gorm:"type:text" json:"content"`
	SenderID   uint     `json:"sender_id"`
	TargetType int      `json:"target_type"` // 1=全员 2=角色 3=指定用户
	CreatedAt  DateTime `json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

// NotificationRecipient 消息收件人（写扩散展开行）
type NotificationRecipient struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	NotificationID uint      `gorm:"index;not null" json:"notification_id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	ReadAt         *DateTime `json:"read_at"` // NULL=未读
}

func (NotificationRecipient) TableName() string { return "notification_recipients" }
