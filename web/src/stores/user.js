import { defineStore } from 'pinia'
import { login as loginApi, getMe } from '../api/auth'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    userInfo: null
  }),
  getters: {
    isLogin: (state) => !!state.token,
    isAdmin: (state) => state.userInfo?.role === 'admin'
  },
  actions: {
    async login(username, password) {
      const data = await loginApi({ username, password })
      this.token = data.token
      this.userInfo = data.user
      localStorage.setItem('token', data.token)
      return data
    },
    async fetchUser() {
      if (!this.token) return
      this.userInfo = await getMe()
    },
    logout() {
      this.token = ''
      this.userInfo = null
      localStorage.removeItem('token')
    }
  }
})
