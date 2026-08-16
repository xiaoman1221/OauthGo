import { md5 } from './md5'

// 头像来源设置取值
export const AVATAR_SOURCE_AUTO = 'auto'
export const AVATAR_SOURCE_QQ = 'qq'
export const AVATAR_SOURCE_GRAVATAR = 'gravatar'

// Gravatar 官方地址与默认国内镜像（Cravatar）
export const OFFICIAL_GRAVATAR_BASE = 'https://www.gravatar.com/avatar'
export const DEFAULT_GRAVATAR_MIRROR = 'https://cravatar.cn/avatar'

export interface AvatarOptions {
  avatar?: string | null
  email?: string | null
  username?: string | null
  size?: number
  /** auto：QQ 邮箱自动用 QQ 头像，其余用 Gravatar；qq：仅 QQ 邮箱用 QQ 头像；gravatar：全部使用 Gravatar */
  source?: string
  /** 是否启用 Gravatar 镜像站（默认开启，面向中国大陆） */
  mirrorEnabled?: boolean
  /** 镜像站基址，形如 https://cravatar.cn/avatar；留空回退官方 gravatar.com */
  mirrorUrl?: string
}

// 头像解析：
// 1. 用户已配置头像地址则直接使用
// 2. 按 avatar_source 设置决定是否使用 QQ 头像（邮箱为数字@qq.com，取 q1.qlogo.cn）
// 3. 其余使用 Gravatar（启用镜像站时走镜像地址，未启用或镜像为空时走官方 gravatar.com）
// 4. 无 Gravatar 头像时使用 identicon 占位图
export function avatarFor(opts: AvatarOptions): string {
  const {
    avatar,
    email,
    username,
    size = 200,
    source = AVATAR_SOURCE_AUTO,
    mirrorEnabled = true,
    mirrorUrl = DEFAULT_GRAVATAR_MIRROR
  } = opts

  if (avatar && avatar.trim()) return avatar.trim()

  const mail = (email || '').trim().toLowerCase()
  const qqMatch = mail.match(/^(\d{5,})@qq\.com$/)

  const useQQ = (source === AVATAR_SOURCE_QQ || source === AVATAR_SOURCE_AUTO) && qqMatch
  if (useQQ && qqMatch) {
    return `https://q1.qlogo.cn/g?b=qq&nk=${qqMatch[1]}&s=${Math.min(Math.max(size, 40), 640)}`
  }

  const seed = mail || username || 'oauthgo'
  const base =
    mirrorEnabled && mirrorUrl && mirrorUrl.trim()
      ? mirrorUrl.trim().replace(/\/+$/, '')
      : OFFICIAL_GRAVATAR_BASE
  return `${base}/${md5(seed)}?d=identicon&s=${size}`
}
