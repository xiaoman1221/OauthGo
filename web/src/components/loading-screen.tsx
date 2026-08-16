export function LoadingScreen({ label = '加载中…' }: { label?: string }) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-background">
      <div className="flex h-9 w-9 items-center justify-center rounded bg-foreground text-sm font-semibold text-background">
        O
      </div>
      <span className="h-4 w-4 animate-spin rounded-full border border-muted-foreground/30 border-t-muted-foreground" />
      <span className="text-sm text-muted-foreground">{label}</span>
    </div>
  )
}
