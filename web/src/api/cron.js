import request from './request'

// 定时任务
export const getCronTasks = (params) => request.get('/cron-tasks', { params })
export const getCronTask = (id) => request.get('/cron-tasks/' + id)
export const addCronTask = (data) => request.post('/cron-tasks', data)
export const updateCronTask = (id, data) => request.put('/cron-tasks/' + id, data)
export const deleteCronTask = (id) => request.delete('/cron-tasks/' + id)
export const updateCronTaskStatus = (id, data) => request.put('/cron-tasks/' + id + '/status', data)
export const runCronTask = (id) => request.post('/cron-tasks/' + id + '/run')
export const getCronTaskExecutions = (id, params) => request.get('/cron-tasks/' + id + '/executions', { params })
