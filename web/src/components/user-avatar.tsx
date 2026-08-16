import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { avatarFor } from '@/lib/avatar'

export function UserAvatar({
  avatar,
  email,
  username,
  size = 'md'
}: {
  avatar?: string | null
  email?: string | null
  username?: string | null
  size?: 'sm' | 'md' | 'lg'
}) {
  const src = avatarFor(avatar, email, username)
  const cls = size === 'sm' ? 'h-6 w-6' : size === 'lg' ? 'h-16 w-16' : 'h-9 w-9'
  return (
    <Avatar className={cls}>
      <AvatarImage src={src} alt={username || 'avatar'} />
      <AvatarFallback>{(username || 'U').slice(0, 1).toUpperCase()}</AvatarFallback>
    </Avatar>
  )
}
