import request from './request'

export const login = (data) => request.post('/auth/login', data)
export const register = (data) => request.post('/auth/register', data)
export const sendCode = (data) => request.post('/auth/send-code', data)
export const forgotPassword = (data) => request.post('/auth/forgot', data)
export const getMe = () => request.get('/auth/me')
export const updateProfile = (data) => request.put('/auth/me', data)
export const changePassword = (data) => request.put('/auth/password', data)
export const myBindings = () => request.get('/auth/bindings')
export const bindLogin = (provider) => request.get(`/auth/bind/${provider}`)
export const unbindLogin = (provider) => request.delete(`/auth/bind/${provider}`)
