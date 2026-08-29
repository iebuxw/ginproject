package service

import (
	"errors"
	"log"
	"sort"

	"ginproject/internal/dao"
	"ginproject/internal/model"
	"ginproject/internal/ws"
)

// SendTarget 发布请求的收件范围
type SendTarget struct {
	TargetType int    `json:"target_type"` // 1=全员 2=角色 3=指定用户
	RoleIDs    []uint `json:"role_ids"`
	UserIDs    []uint `json:"user_ids"`
}

type NotificationService struct {
	notificationDAO *dao.NotificationDAO
	userDAO         *dao.UserDAO
	hub             *ws.Hub
}

func NewNotificationService(nd *dao.NotificationDAO, ud *dao.UserDAO, hub *ws.Hub) *NotificationService {
	return &NotificationService{notificationDAO: nd, userDAO: ud, hub: hub}
}

// Send 发布消息并推送在线收件人
func (s *NotificationService) Send(n *model.Notification, target SendTarget) error {
	userIDs, err := expandTargets(target)
	if err != nil {
		return err
	}
	if err := s.notificationDAO.CreateWithRecipients(n, userIDs); err != nil {
		return err
	}
	// 推送在线用户（离线静默，属正常情况）
	for _, uid := range userIDs {
		s.hub.Send(uid, ws.Message{
			Type:    "notification",
			ID:      n.ID,
			Title:   n.Title,
			Content: n.Content,
		})
	}
	return nil
}

// SendSystemEvent 系统事件落库（worker 调用；落库失败不阻断业务流程）
func (s *NotificationService) SendSystemEvent(title, content string, userID uint) {
	n := &model.Notification{
		Type:       3,
		Title:      title,
		Content:    content,
		SenderID:   0,
		TargetType: 3,
	}
	if err := s.notificationDAO.CreateWithRecipients(n, []uint{userID}); err != nil {
		log.Printf("系统事件消息落库失败: %v", err)
		return
	}
	s.hub.Send(userID, ws.Message{
		Type:    "notification",
		ID:      n.ID,
		Title:   title,
		Content: content,
	})
}

func (s *NotificationService) Delete(id uint) error { return s.notificationDAO.Delete(id) }

func (s *NotificationService) FindPage(page, pageSize int, keyword string) ([]model.Notification, int64, error) {
	return s.notificationDAO.FindPage(page, pageSize, keyword)
}

func (s *NotificationService) FindUserPage(userID uint, page, pageSize, readStatus, notifType int) ([]dao.NotificationItem, int64, error) {
	return s.notificationDAO.FindUserPage(userID, page, pageSize, readStatus, notifType)
}

func (s *NotificationService) MarkRead(userID uint, ids []uint) error {
	return s.notificationDAO.MarkRead(userID, ids)
}

func (s *NotificationService) UnreadCount(userID uint) (int64, error) {
	return s.notificationDAO.UnreadCount(userID)
}

// expandTargets 校验收件范围并展开为去重排序的 user_id 列表。
// 全员广播由调用方传入 userIDs=全部用户 id，此处统一做校验与去重。
func expandTargets(target SendTarget) ([]uint, error) {
	switch target.TargetType {
	case 1:
		if len(target.UserIDs) == 0 {
			return nil, errors.New("全员广播收件人为空")
		}
	case 2, 3:
		if len(target.UserIDs) == 0 {
			return nil, errors.New("收件人不能为空")
		}
	default:
		return nil, errors.New("无效的收件范围")
	}
	seen := make(map[uint]struct{}, len(target.UserIDs))
	result := make([]uint, 0, len(target.UserIDs))
	for _, id := range target.UserIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}
