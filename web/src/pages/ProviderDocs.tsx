import { useState } from 'react'
import { ExternalLink } from 'lucide-react'
import { PageHeader } from '@/components/page-header'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { channelListTyped } from '@/lib/providerDocs'
import { cn } from '@/lib/utils'

export default function ProviderDocs() {
  const [active, setActive] = useState(channelListTyped[0]?.name || '')

  const current = channelListTyped.find((c) => c.name === active) || channelListTyped[0]

  return (
    <div>
      <PageHeader
        title="第三方渠道接入"
        description="各第三方登录渠道的申请步骤、回调配置与注意事项。配置完成后在「登录渠道」中填入凭据。"
      />

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-4">
        {/* 渠道索引 */}
        <aside className="lg:sticky lg:top-20 lg:self-start">
          <div className="flex flex-wrap gap-1 lg:flex-col">
            {channelListTyped.map((c) => (
              <button
                key={c.name}
                onClick={() => c.name && setActive(c.name)}
                className={cn(
                  'flex items-center justify-between gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors duration-150',
                  active === c.name ? 'bg-accent font-medium text-foreground' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
                )}
              >
                <span>{c.displayName}</span>
                <span className="text-xs opacity-60">{c.name}</span>
              </button>
            ))}
          </div>
        </aside>

        {/* 文档内容 */}
        {current && (
          <article className="lg:col-span-3">
            <div className="flex items-center gap-3">
              <h2 className="text-2xl font-light tracking-tighter">{current.displayName}</h2>
              <Badge variant={current.category === 'enterprise' ? 'warning' : 'secondary'} className="font-normal">
                {current.category === 'enterprise' ? '企业' : '社交'}
              </Badge>
              <span className="font-mono text-xs text-muted-foreground">{current.name}</span>
            </div>

            <p className="mt-4 max-w-2xl text-sm leading-relaxed text-muted-foreground">{current.tips}</p>

            {current.registerUrl && (
              <a
                href={current.registerUrl}
                target="_blank"
                rel="noreferrer"
                className="mt-3 inline-flex items-center gap-1.5 text-sm font-medium text-foreground underline-offset-4 hover:underline"
              >
                {current.registerLabel} <ExternalLink className="h-3.5 w-3.5" />
              </a>
            )}

            <Separator className="my-8" />

            <h3 className="text-sm font-medium uppercase tracking-wider text-muted-foreground">接入步骤</h3>
            <ol className="mt-4 space-y-3">
              {current.steps.map((step, i) => (
                <li key={i} className="flex gap-3 text-sm leading-relaxed">
                  <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-border text-xs text-muted-foreground">
                    {i + 1}
                  </span>
                  <span>{step}</span>
                </li>
              ))}
            </ol>

            <Separator className="my-8" />

            <h3 className="text-sm font-medium uppercase tracking-wider text-muted-foreground">回调配置</h3>
            <div className="mt-4 rounded-md border border-border bg-muted/30 p-4 text-sm leading-relaxed">
              {current.callbackNote || '无需配置回调地址（前端直接传 code 登录）。'}
            </div>

            {current.notes.length > 0 && (
              <>
                <Separator className="my-8" />
                <h3 className="text-sm font-medium uppercase tracking-wider text-muted-foreground">注意事项</h3>
                <ul className="mt-4 list-disc space-y-2 pl-5 text-sm leading-relaxed text-muted-foreground">
                  {current.notes.map((n, i) => (
                    <li key={i}>{n}</li>
                  ))}
                </ul>
              </>
            )}
          </article>
        )}
      </div>
    </div>
  )
}
