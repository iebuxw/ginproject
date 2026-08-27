import request from './request'

export const getUsers = (params) => request.get('/users', { params })
export const getUser = (id) => request.get('/users/' + id)
export const addUser = (data) => request.post('/users', data)
export const updateUser = (id, data) => request.put('/users/' + id, data)
export const deleteUser = (id) => request.delete('/users/' + id)
export const uploadAvatar = (formData) => request.post('/upload/avatar', formData, {
  headers: { 'Content-Type': 'multipart/form-data' }
})
