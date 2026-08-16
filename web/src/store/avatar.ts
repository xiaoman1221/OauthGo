import { create } from 'zustand'
import { settingsApi } from '@/lib/api'
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

export const useAvatarStore = create<AvatarState>((set) => ({
  settings: defaults,
  loaded: false,
  load: async () => {
    try {
      const data = await settingsApi.list()
      const group = (data.groups && data.groups['avatar']) || []
      const m: Record<string, string> = {}
      group.forEach((item) => {
        m[item.key] = item.value
      })
      const mirror = (m.gravatar_mirror || '').trim()
      set({
        settings: {
          source: m.avatar_source || defaults.source,
          mirrorEnabled: m.gravatar_mirror_enabled === '' ? defaults.mirrorEnabled : m.gravatar_mirror_enabled !== '0',
          mirrorUrl: mirror || defaults.mirrorUrl
        },
        loaded: true
      })
    } catch {
      set({ loaded: true })
    }
  }
}))
