import request from './request'

export const getLogSettings = () => request.get('/log-settings')
export const updateLogSettings = (data) => request.put('/log-settings', data)
