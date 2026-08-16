import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Download, Search, Trash2 } from 'lucide-react'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ui/confirm'
import { loginsApi, type LoginRecord } from '@/lib/api'

export default function Logins() {
  const [list, setList] = useState<LoginRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('')
  const [selected, setSelected] = useState<number[]>([])
  const [detail, setDetail] = useState<LoginRecord | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<LoginRecord | null>(null)
  const [batchConfirm, setBatchConfirm] = useState(false)
  const [loading, setLoading] = useState(false)

  const load = useCallback(
    async (p = page) => {
      setLoading(true)
      try {
        const params: Record<string, unknown> = { page: p, page_size: pageSize }
        if (keyword.trim()) params.keyword = keyword.trim()
        if (status) params.status = status
        const data = await loginsApi.list(params)
        setList(data.list)
        setTotal(data.total)
        setPage(p)
        setSelected([])
      } catch (err) {
        toast.error((err as Error).message)
      } finally {
        setLoading(false)
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [keyword, status, pageSize]
  )

  useEffect(() => {
    load(1)
  }, [load])

  const onDelete = async () => {
    if (!deleteTarget) return
    try {
      await loginsApi.remove(deleteTarget.id)
      toast.success('删除成功')
      setDeleteTarget(null)
      load(page)
    } catch (err) {
      toast.error((err as Error).message)
    }
  }

  const onBatchDelete = async () => {
    try {
      await loginsApi.batchRemove(selected)
      toast.success('批量删除成功')
      setBatchConfirm(false)
      load(page)
    } catch (err) {
      toast.error((err as Error).message)
    }
  }

  const onExport = async () => {
    try {
      const res = await loginsApi.export({ keyword: keyword.trim(), status: status || undefined })
      const blob = res as unknown as Blob
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'login_records.csv'
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      toast.error((err as Error).message)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div>
      <PageHeader
        title="登录记录"
        description="目标站点通过本平台完成的第三方登录行为记录。"
        actions={
          <Button variant="outline" onClick={onExport}>
            <Download className="h-4 w-4" /> 导出 CSV
          </Button>
        }
      />

      {/* 筛选栏 */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && load(1)}
            placeholder="搜索用户名 / 昵称 / IP"
            className="w-64 pl-8"
          />
        </div>
        <Select value={status} onValueChange={(v) => setStatus(v === 'all' ? '' : v)}>
          <SelectTrigger className="w-32">
            <SelectValue placeholder="全部状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="1">成功</SelectItem>
            <SelectItem value="0">失败</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="ghost" onClick={() => { setKeyword(''); setStatus(''); load(1) }}>
          重置
        </Button>
        {selected.length > 0 && (
          <Button variant="outline" className="text-destructive" onClick={() => setBatchConfirm(true)}>
            <Trash2 className="h-4 w-4" /> 删除所选（{selected.length}）
          </Button>
        )}
      </div>

      <div className="overflow-hidden rounded-md border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                <input
                  type="checkbox"
                  className="accent-foreground"
                  checked={list.length > 0 && selected.length === list.length}
                  onChange={(e) => setSelected(e.target.checked ? list.map((r) => r.id) : [])}
                />
              </TableHead>
              <TableHead className="w-14">ID</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>应用</TableHead>
              <TableHead>平台</TableHead>
              <TableHead>IP</TableHead>
              <TableHead className="w-40">登录时间</TableHead>
              <TableHead className="w-16">状态</TableHead>
              <TableHead className="w-24 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {!loading && list.length === 0 && (
              <TableRow>
                <TableCell colSpan={9} className="h-24 text-center text-sm text-muted-foreground">
                  暂无登录记录
                </TableCell>
              </TableRow>
            )}
            {list.map((r) => (
              <TableRow key={r.id}>
                <TableCell>
                  <input
                    type="checkbox"
                    className="accent-foreground"
                    checked={selected.includes(r.id)}
                    onChange={(e) =>
                      setSelected(e.target.checked ? [...selected, r.id] : selected.filter((id) => id !== r.id))
                    }
                  />
                </TableCell>
                <TableCell className="text-muted-foreground">{r.id}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-2.5">
                    <Avatar className="h-7 w-7">
                      <AvatarImage src={r.avatar} alt={r.nickname} />
                      <AvatarFallback>{(r.nickname || r.username || 'U').slice(0, 1)}</AvatarFallback>
                    </Avatar>
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium">{r.nickname || r.username || '-'}</div>
                      {r.uid_label && <div className="truncate text-xs text-muted-foreground">{r.uid_label} · {r.uid_value}</div>}
                    </div>
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground">{r.app_name || 'NULL'}</TableCell>
                <TableCell className="text-muted-foreground">{r.platform || '-'}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{r.ip}</TableCell>
                <TableCell className="text-xs text-muted-foreground">{r.login_time}</TableCell>
                <TableCell>
                  <StatusBadge status={r.status === 1 ? 'success' : 'danger'}>{r.status === 1 ? '成功' : '失败'}</StatusBadge>
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <Button variant="ghost" size="sm" onClick={() => setDetail(r)}>
                      详情
                    </Button>
                    <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setDeleteTarget(r)}>
                      删除
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* 分页 */}
      <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
        <span>
          共 {total} 条记录
        </span>
        <div className="flex items-center gap-1">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => load(page - 1)}>
            上一页
          </Button>
          <span className="px-3 text-xs">
            {page} / {totalPages}
          </span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => load(page + 1)}>
            下一页
          </Button>
        </div>
      </div>

      {/* 详情 */}
      <Dialog open={!!detail} onOpenChange={(open) => !open && setDetail(null)}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>登录记录详情</DialogTitle>
            <DialogDescription>ID {detail?.id} · {detail?.login_time}</DialogDescription>
          </DialogHeader>
          {detail && (
            <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
              <DetailItem label="应用" value={detail.app_name || 'NULL'} />
              <DetailItem label="平台" value={detail.platform || '-'} />
              <div className="col-span-2">
                <dt className="text-xs text-muted-foreground">用户</dt>
                <dd className="mt-0.5">
                  {detail.nickname || detail.username || '-'}
                  {detail.uid_label && (
                    <span className="ml-2 text-xs text-muted-foreground">
                      {detail.uid_label} · {detail.uid_value}
                    </span>
                  )}
                </dd>
              </div>
              <DetailItem label="IP" value={detail.ip || '-'} mono />
              <DetailItem label="归属地" value={detail.location || '-'} />
              <div>
                <dt className="text-xs text-muted-foreground">状态</dt>
                <dd className="mt-1">
                  <StatusBadge status={detail.status === 1 ? 'success' : 'danger'}>{detail.status === 1 ? '成功' : '失败'}</StatusBadge>
                </dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">头像</dt>
                <dd className="mt-1">
                  {detail.avatar ? (
                    <Avatar className="h-10 w-10 rounded">
                      <AvatarImage src={detail.avatar} alt={detail.nickname} />
                      <AvatarFallback>{(detail.nickname || 'U').slice(0, 1)}</AvatarFallback>
                    </Avatar>
                  ) : (
                    <span className="text-muted-foreground">-</span>
                  )}
                </dd>
              </div>
              <div className="col-span-2">
                <dt className="text-xs text-muted-foreground">User-Agent</dt>
                <dd className="mt-0.5 break-all text-xs text-muted-foreground">{detail.user_agent || '-'}</dd>
              </div>
            </dl>
          )}
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="确定删除该条记录吗？"
        confirmText="删除"
        destructive
        onConfirm={onDelete}
      />
      <ConfirmDialog
        open={batchConfirm}
        onOpenChange={setBatchConfirm}
        title={`确定删除选中的 ${selected.length} 条记录吗？`}
        confirmText="删除"
        destructive
        onConfirm={onBatchDelete}
      />
    </div>
  )
}

function DetailItem({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className={`mt-0.5 ${mono ? 'font-mono text-xs' : ''}`}>{value}</dd>
    </div>
  )
}
