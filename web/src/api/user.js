import request from './request'

export const getUsers = (params) => request.get('/users', { params })
export const getUser = (id) => request.get('/users/' + id)
export const addUser = (data) => request.post('/users', data)
export const updateUser = (id, data) => request.put('/users/' + id, data)
export const deleteUser = (id) => request.delete('/users/' + id)
export const uploadAvatar = (formData) => request.post('/upload/avatar', formData, {
  headers: { 'Content-Type': 'multipart/form-data' }
})
export const exportUsers = (params) => request.get('/users/export', { params, responseType: 'blob' })
export const importUsers = (formData) => request.post('/users/import', formData, {
  headers: { 'Content-Type': 'multipart/form-data' }
})
export const downloadImportTemplate = () => request.get('/users/import-template', { responseType: 'blob' })
