import { cn } from '@/lib/utils'

type Status = 'success' | 'danger' | 'warning' | 'muted' | 'default'

const styles: Record<Status, string> = {
  success: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
  danger: 'border-red-500/20 bg-red-500/10 text-red-700 dark:text-red-400',
  warning: 'border-amber-500/20 bg-amber-500/10 text-amber-700 dark:text-amber-400',
  muted: 'border-zinc-500/20 bg-zinc-500/10 text-zinc-600 dark:text-zinc-400',
  default: 'border-foreground/15 bg-foreground/5 text-foreground'
}

const dots: Record<Status, string> = {
  success: 'bg-emerald-500',
  danger: 'bg-red-500',
  warning: 'bg-amber-500',
  muted: 'bg-zinc-400 dark:bg-zinc-500',
  default: 'bg-foreground'
}

// 椭圆形状态胶囊：圆点 + 文案，背景为柔和的半透明椭圆色块
export function StatusBadge({
  status,
  children,
  className
}: {
  status: Status
  children: React.ReactNode
  className?: string
}) {
  return (
    <span
      className={cn(
        'inline-flex h-[22px] items-center gap-1.5 whitespace-nowrap rounded-full border px-2.5 text-xs font-medium',
        styles[status],
        className
      )}
    >
      <span className={cn('h-1.5 w-1.5 shrink-0 rounded-full', dots[status])} />
      {children}
    </span>
  )
}
