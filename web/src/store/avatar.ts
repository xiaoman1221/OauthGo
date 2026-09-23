import { create } from 'zustand'
import { authApi } from '@/lib/api'
import { DEFAULT_GRAVATAR_MIRROR, AVATAR_SOURCE_AUTO } from '@/lib/avatar'

export interface AvatarSettings {
  source: string
  mirrorEnabled: boolean
  mirrorUrl: string
}

interface AvatarState {
  settings: AvatarSettings
  loaded: boolean
  load: () => Promise<void>
}

const defaults: AvatarSettings = {
  source: AVATAR_SOURCE_AUTO,
  mirrorEnabled: true,
  mirrorUrl: DEFAULT_GRAVATAR_MIRROR
}

// 头像设置为公开只读配置，走 /api/auth/config（系统设置接口已收归管理员）
export const useAvatarStore = create<AvatarState>((set) => ({
  settings: defaults,
  loaded: false,
  load: async () => {
    try {
      const cfg = await authApi.config()
      const mirror = String(cfg.gravatar_mirror ?? '').trim()
      const mirrorEnabledRaw = String(cfg.gravatar_mirror_enabled ?? '')
      set({
        settings: {
          source: (cfg.avatar_source as string) || defaults.source,
          mirrorEnabled: mirrorEnabledRaw === '' ? defaults.mirrorEnabled : mirrorEnabledRaw !== '0',
          mirrorUrl: mirror || defaults.mirrorUrl
        },
        loaded: true
      })
    } catch {
      set({ loaded: true })
    }
  }
}))
