import { create } from 'zustand'
import { authApi } from '@/lib/api'
import type { User } from '@/lib/api'

interface UserState {
  token: string
  userInfo: User | null
  login: (username: string, password: string) => Promise<User>
  setToken: (token: string) => void
  fetchUser: () => Promise<void>
  setUserInfo: (user: User) => void
  logout: () => void
}

export const useUserStore = create<UserState>((set) => ({
  token: localStorage.getItem('token') || '',
  userInfo: null,
  login: async (username, password) => {
    const data = await authApi.login({ username, password })
    localStorage.setItem('token', data.token)
    set({ token: data.token, userInfo: data.user })
    return data.user
  },
  setToken: (token) => {
    localStorage.setItem('token', token)
    set({ token })
  },
  fetchUser: async () => {
    const token = localStorage.getItem('token')
    if (!token) return
    const user = await authApi.me()
    set({ userInfo: user })
  },
  setUserInfo: (user) => set({ userInfo: user }),
  logout: () => {
    localStorage.removeItem('token')
    set({ token: '', userInfo: null })
  }
}))

// 派生选择器
export const useIsLogin = () => useUserStore((s) => !!s.token)
export const useIsAdmin = () => useUserStore((s) => s.userInfo?.role === 'admin')
