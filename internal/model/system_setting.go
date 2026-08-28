package model

type SystemSetting struct {
	ID           int64    `json:"id" gorm:"primaryKey;autoIncrement"`
	SettingKey   string   `json:"setting_key" gorm:"size:64;not null;uniqueIndex"`
	SettingValue string   `json:"setting_value" gorm:"type:text"`
	CreatedAt    DateTime `json:"created_at"`
	UpdatedAt    DateTime `json:"updated_at"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}
