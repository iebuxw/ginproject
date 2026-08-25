import request from './request'

// 字典类型
export const getDictTypes = (params) => request.get('/dict-types', { params })
export const getDictType = (id) => request.get('/dict-types/' + id)
export const addDictType = (data) => request.post('/dict-types', data)
export const updateDictType = (id, data) => request.put('/dict-types/' + id, data)
export const deleteDictType = (id) => request.delete('/dict-types/' + id)

// 字典数据
export const getDictData = (params) => request.get('/dict-data', { params })
export const getDictDataById = (id) => request.get('/dict-data/' + id)
export const getDictDataByCode = (code) => request.get('/dict-data/by-code/' + code)
export const addDictData = (data) => request.post('/dict-data', data)
export const updateDictData = (id, data) => request.put('/dict-data/' + id, data)
export const deleteDictData = (id) => request.delete('/dict-data/' + id)
