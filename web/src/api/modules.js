import request from './request'

// 应用管理
export const listApps = () => request.get('/apps')
export const createApp = (data) => request.post('/apps', data)
export const updateApp = (id, data) => request.put(`/apps/${id}`, data)
export const deleteApp = (id) => request.delete(`/apps/${id}`)

// 登录记录
export const listLogins = (params) => request.get('/logins', { params })
export const deleteLogin = (id) => request.delete(`/logins/${id}`)
export const batchDeleteLogins = (ids) => request.post('/logins/batch-delete', { ids })
export const exportLogins = (params) =>
  request.get('/logins/export', { params, responseType: 'blob' })
// importLogins removed: importing login records is disabled


// 系统设置
export const getSettings = () => request.get('/settings')
export const updateSettings = (items) => request.put('/settings', { items })
export const testSMTP = (to) => request.post('/settings/test/smtp', { to })
export const testSMS = (phone) => request.post('/settings/test/sms', { phone })

// 用户管理
export const listUsers = (params) => request.get('/users', { params })
export const createUser = (data) => request.post('/users', data)
export const updateUser = (id, data) => request.put(`/users/${id}`, data)
export const deleteUser = (id) => request.delete(`/users/${id}`)

// 第三方登录渠道
export const publicProviders = () => request.get('/oauth/providers')
export const listProviders = () => request.get('/providers')
export const updateProvider = (name, data) => request.put(`/providers/${name}`, data)
export const testProvider = (name, data) => request.post(`/providers/${name}/test`, data)
