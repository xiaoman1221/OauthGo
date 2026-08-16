import { useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

export function CodeBlock({ code, className }: { code: string; className?: string }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      toast.success('已复制')
      setTimeout(() => setCopied(false), 1500)
    } catch {
      toast.warning('复制失败，请手动复制')
    }
  }
  return (
    <div className={cn('group relative', className)}>
      <button
        onClick={copy}
        className="absolute right-2 top-2 rounded p-1 text-muted-foreground opacity-0 transition-opacity duration-150 group-hover:opacity-100 hover:bg-muted"
        aria-label="复制"
      >
        {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      </button>
      <pre className="overflow-x-auto rounded-md border border-border bg-muted/40 px-4 py-3 text-xs leading-relaxed text-foreground">
        <code>{code}</code>
      </pre>
    </div>
  )
}
