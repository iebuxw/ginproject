import request from './request'

export const getLogs = (params) => request.get('/logs', { params })

export const getLoginLogs = (params) => request.get('/login-logs', { params })
