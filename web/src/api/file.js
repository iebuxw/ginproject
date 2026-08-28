import request from './request'

export const getFiles = (params) => request.get('/files', { params })

export const uploadFile = (formData) => request.post('/files/upload', formData, { timeout: 0 })

export const deleteFile = (id) => request.delete(`/files/${id}`)
