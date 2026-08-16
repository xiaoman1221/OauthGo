import axios from 'axios'
import type { AxiosRequestConfig } from 'axios'

const http = axios.create({
  baseURL: '/api',
  timeout: 20000
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res && typeof res === 'object' && 'code' in res && res.code !== 0) {
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return res
  },
  (error) => {
    if (error.response && error.response.status === 401) {
      localStorage.removeItem('token')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    const message = error.response?.data?.message || error.message || '网络错误'
    return Promise.reject(new Error(message))
  }
)

// 类型定义
export interface User {
  id: number
  username: string
  nickname: string
  avatar: string
  email: string
  phone: string
  role: string
  password_set: boolean
  created_at: string
  updated_at: string
}

export interface App {
  id: number
  owner_id: number
  name: string
  platform: string
  appid: string
  app_key: string
  mode: string
  types: string[]
  domains: string
  status: number
  created_at: string
  updated_at: string
}

export interface LoginRecord {
  id: number
  app_id: number
  app_name: string
  open_id: string
  username: string
  nickname: string
  avatar: string
  platform: string
  ip: string
  location: string
  user_agent: string
  status: number
  login_time: string
  uid_label: string
  uid_value: string
}

export interface Provider {
  id: number
  name: string
  display_name: string
  category: string
  client_id: string
  client_secret: string
  config: string
  enabled: boolean
  main_site: boolean
  sort: number
  callback_url: string
}

export interface SettingDef {
  key: string
  value: string
  description: string
  group: string
  sensitive: boolean
}

export interface Binding {
  name: string
  display_name: string
  category: string
  bound: boolean
  nickname?: string
  avatar?: string
}

async function get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  const res = await http.get(url, config)
  return (res as unknown as { data: T }).data
}

async function post<T>(url: string, data?: unknown): Promise<T> {
  const res = await http.post(url, data)
  return (res as unknown as { data: T }).data
}

async function put<T>(url: string, data?: unknown): Promise<T> {
  const res = await http.put(url, data)
  return (res as unknown as { data: T }).data
}

async function del<T>(url: string): Promise<T> {
  const res = await http.delete(url)
  return (res as unknown as { data: T }).data
}

// ---------- 认证 ----------
export const authApi = {
  config: () => get<Record<string, unknown>>('/auth/config'),
  login: (data: { username: string; password: string }) =>
    post<{ token: string; user: User }>('/auth/login', data),
  register: (data: Record<string, unknown>) => post<unknown>('/auth/register', data),
  sendCode: (data: { scope: string; account: string }) => post<unknown>('/auth/send-code', data),
  forgot: (data: { account: string; code: string; password: string }) =>
    post<unknown>('/auth/forgot', data),
  me: () => get<User>('/auth/me'),
  updateProfile: (data: Record<string, unknown>) => put<User>('/auth/me', data),
  changePassword: (data: { old_password?: string; new_password: string }) =>
    put<unknown>('/auth/password', data),
  bindings: () => get<Binding[]>('/auth/bindings'),
  bindLogin: (provider: string) => get<{ url: string }>(`/auth/bind/${provider}`),
  unbindLogin: (provider: string) => del<unknown>(`/auth/bind/${provider}`)
}

// ---------- 应用 ----------
export const appsApi = {
  list: () => get<{ list: App[] }>('/apps'),
  create: (data: Record<string, unknown>) => post<App>('/apps', data),
  update: (id: number, data: Record<string, unknown>) => put<App>(`/apps/${id}`, data),
  remove: (id: number) => del<unknown>(`/apps/${id}`)
}

// ---------- 登录记录 ----------
export const loginsApi = {
  list: (params: Record<string, unknown>) =>
    get<{ list: LoginRecord[]; total: number }>('/logins', { params }),
  remove: (id: number) => del<unknown>(`/logins/${id}`),
  batchRemove: (ids: number[]) => post<unknown>('/logins/batch-delete', { ids }),
  export: (params: Record<string, unknown>) =>
    http.get('/logins/export', { params, responseType: 'blob' })
}

// ---------- 设置 ----------
export const settingsApi = {
  list: () => get<{ groups: Record<string, SettingDef[]> }>('/settings'),
  update: (items: { key: string; value: string }[]) => put<unknown>('/settings', { items }),
  testSMTP: (to: string) => post<unknown>('/settings/test/smtp', { to }),
  testSMS: (phone: string) => post<unknown>('/settings/test/sms', { phone })
}

// ---------- 用户管理 ----------
export const usersApi = {
  list: (params: Record<string, unknown>) =>
    get<{ list: User[]; total: number }>('/users', { params }),
  create: (data: Record<string, unknown>) => post<User>('/users', data),
  update: (id: number, data: Record<string, unknown>) => put<User>(`/users/${id}`, data),
  remove: (id: number) => del<unknown>(`/users/${id}`)
}

// ---------- 登录渠道 ----------
export const providersApi = {
  public: () => get<{ name: string; display_name: string; category: string }[]>('/oauth/providers'),
  list: () => get<{ list: Provider[] }>('/providers'),
  update: (name: string, data: Record<string, unknown>) => put<unknown>(`/providers/${name}`, data),
  test: (name: string, data: Record<string, unknown>) =>
    post<{ message: string; auth_url?: string }>(`/providers/${name}/test`, data)
}

export { http }
