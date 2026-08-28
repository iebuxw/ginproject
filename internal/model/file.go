package model

type File struct {
	ID           int64    `json:"id" gorm:"primaryKey;autoIncrement"`
	OriginalName string   `json:"original_name" gorm:"size:255;not null"`
	StoredName   string   `json:"stored_name" gorm:"size:128;not null"`
	Size         int64    `json:"size"`
	Ext          string   `json:"ext" gorm:"size:32"`
	UploaderID   int64    `json:"uploader_id"`
	UploaderName string   `json:"uploader_name" gorm:"size:64"`
	CreatedAt    DateTime `json:"created_at"`
}

func (File) TableName() string {
	return "files"
}
