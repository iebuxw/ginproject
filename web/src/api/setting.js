import request from './request'

export const getSettings = () => request.get('/settings')
export const updateSettings = (data) => request.put('/settings', data)
export const uploadLogo = (formData) => request.post('/upload/logo', formData, {
  headers: { 'Content-Type': 'multipart/form-data' },
  timeout: 0
})
