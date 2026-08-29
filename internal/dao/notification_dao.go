package dao

import (
	"time"

	"ginproject/internal/model"

	"gorm.io/gorm"
)

// NotificationItem 用户视角的消息行（join 结果）
type NotificationItem struct {
	ID         uint            `json:"id"`
	Type       int             `json:"type"`
	Title      string          `json:"title"`
	Content    string          `json:"content"`
	SenderID   uint            `json:"sender_id"`
	SenderName string          `json:"sender_name"` // 发布人用户名；系统事件（sender_id=0）为空
	CreatedAt  model.DateTime  `json:"created_at"`
	ReadAt     *model.DateTime `json:"read_at"`
}

type NotificationDAO struct{ db *gorm.DB }

func NewNotificationDAO(db *gorm.DB) *NotificationDAO {
	return &NotificationDAO{db: db}
}

// CreateWithRecipients 事务写入消息本体+收件人
func (d *NotificationDAO) CreateWithRecipients(n *model.Notification, userIDs []uint) error {
	// DateTime 无 default tag，GORM 会把零值显式写入 created_at 列（绕过 DB DEFAULT CURRENT_TIMESTAMP），必须手动赋值
	if time.Time(n.CreatedAt).IsZero() {
		n.CreatedAt = model.DateTime(time.Now())
	}
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(n).Error; err != nil {
			return err
		}
		if len(userIDs) == 0 {
			return nil
		}
		recipients := make([]model.NotificationRecipient, len(userIDs))
		for i, uid := range userIDs {
			recipients[i] = model.NotificationRecipient{
				NotificationID: n.ID,
				UserID:         uid,
			}
		}
		return tx.Create(&recipients).Error
	})
}

// Delete 连带删除收件记录
func (d *NotificationDAO) Delete(id uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("notification_id = ?", id).Delete(&model.NotificationRecipient{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Notification{}, id).Error
	})
}

// FindPage 管理端分页（发布人视角）
func (d *NotificationDAO) FindPage(page, pageSize int, keyword string) ([]model.Notification, int64, error) {
	var list []model.Notification
	var total int64
	q := d.db.Model(&model.Notification{})
	if keyword != "" {
		q = q.Where("title LIKE ?", "%"+keyword+"%")
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&list).Error
	return list, total, err
}

// FindUserPage 用户视角分页：join 收件人表，readStatus 0=全部 1=未读 2=已读；type 0=全部
func (d *NotificationDAO) FindUserPage(userID uint, page, pageSize, readStatus, notifType int) ([]NotificationItem, int64, error) {
	var list []NotificationItem
	var total int64
	q := d.db.Table("notifications n").
		Select("n.id, n.type, n.title, n.content, n.sender_id, n.created_at, r.read_at, u.username AS sender_name").
		Joins("JOIN notification_recipients r ON r.notification_id = n.id AND r.user_id = ?", userID).
		Joins("LEFT JOIN users u ON u.id = n.sender_id")
	if readStatus == 1 {
		q = q.Where("r.read_at IS NULL")
	} else if readStatus == 2 {
		q = q.Where("r.read_at IS NOT NULL")
	}
	if notifType > 0 {
		q = q.Where("n.type = ?", notifType)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Order("n.id DESC").Find(&list).Error
	return list, total, err
}

// MarkRead 批量已读：置 read_at（仅当前用户自己的记录）
func (d *NotificationDAO) MarkRead(userID uint, ids []uint) error {
	now := model.DateTime(time.Now())
	q := d.db.Model(&model.NotificationRecipient{}).
		Where("user_id = ? AND read_at IS NULL", userID)
	if len(ids) > 0 {
		q = q.Where("notification_id IN ?", ids)
	}
	return q.Update("read_at", now).Error
}

// UnreadCount 当前用户未读数
func (d *NotificationDAO) UnreadCount(userID uint) (int64, error) {
	var count int64
	err := d.db.Model(&model.NotificationRecipient{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return count, err
}
