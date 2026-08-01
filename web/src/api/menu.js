import request from './request'

export const getMenus = () => request.get('/menus')
export const addMenu = (data) => request.post('/menus', data)
export const updateMenu = (id, data) => request.put('/menus/' + id, data)
export const deleteMenu = (id) => request.delete('/menus/' + id)
