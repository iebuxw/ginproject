import request from './request'

export const login = (data) => request.post('/auth/login', data)
export const logout = () => request.post('/auth/logout')
export const getUserInfo = () => request.get('/auth/userinfo')
export const changePassword = (data) => request.post('/auth/change-password', data)
export const updateProfile = (data) => request.put('/auth/profile', data)
