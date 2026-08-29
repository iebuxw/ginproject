import request from './request'

// 发布消息
export const sendNotification = (data) => request.post('/notifications', data)
// 管理端列表
export const getNotifications = (params) => request.get('/notifications', { params })
// 删除消息
export const deleteNotification = (id) => request.delete('/notifications/' + id)
// 我的消息
export const getMyNotifications = (params) => request.get('/notifications/mine', { params })
// 标记已读
export const markRead = (data) => request.post('/notifications/read', data)
// 未读数
export const getUnreadCount = () => request.get('/notifications/unread-count')
