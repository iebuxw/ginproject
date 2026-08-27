import request from './request'

export const getBackups = (params) => request.get('/db-backups', { params })

export const createBackup = () => request.post('/db-backups')

export const restoreBackup = (id) => request.post(`/db-backups/${id}/restore`)

export const deleteBackup = (id) => request.delete(`/db-backups/${id}`)

export const downloadBackup = (id) => request.get(`/db-backups/${id}/download`, { responseType: 'blob' })
