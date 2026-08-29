import request from './request'

export const getRoles = (params) => request.get('/roles', { params })
export const exportRoles = (params) => request.get('/roles/export', { params, responseType: 'blob' })
export const getRole = (id) => request.get('/roles/' + id)
export const addRole = (data) => request.post('/roles', data)
export const updateRole = (id, data) => request.put('/roles/' + id, data)
export const deleteRole = (id) => request.delete('/roles/' + id)
