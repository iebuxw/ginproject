import request from './request'

export const getLogs = (params) => request.get('/logs', { params })
