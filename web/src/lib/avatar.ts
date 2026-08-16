import { md5 } from './md5'

// 头像解析：
// 1. 用户已配置头像地址则直接使用
// 2. 邮箱为 QQ 邮箱（数字@qq.com）时使用 QQ 头像（q1.qlogo.cn）
// 3. 否则回退到 Gravatar（无头像时使用 identicon 占位图）
export function avatarFor(avatar?: string | null, email?: string | null, username?: string | null, size = 200): string {
  if (avatar && avatar.trim()) return avatar.trim()

  const mail = (email || '').trim().toLowerCase()
  const qqMatch = mail.match(/^(\d{5,})@qq\.com$/)
  if (qqMatch) {
    return `https://q1.qlogo.cn/g?b=qq&nk=${qqMatch[1]}&s=${Math.min(Math.max(size, 40), 640)}`
  }

  const seed = mail || username || 'oauthgo'
  return `https://www.gravatar.com/avatar/${md5(seed)}?d=identicon&s=${size}`
}
