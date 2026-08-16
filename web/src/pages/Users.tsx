import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Plus } from 'lucide-react'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { UserAvatar } from '@/components/user-avatar'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
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
import { usersApi, type User } from '@/lib/api'

interface UserForm {
  id?: number
  username: string
  email: string
  role: string
  password: string
}

export default function Users() {
  const [users, setUsers] = useState<User[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [form, setForm] = useState<UserForm>({ username: '', email: '', role: 'user', password: '' })
  const [saving, setSaving] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)

  const load = useCallback(async (p = 1) => {
    try {
      const data = await usersApi.list({ page: p, page_size: 20 })
      setUsers(data.list)
      setTotal(data.total)
      setPage(p)
    } catch (err) {
      toast.error((err as Error).message)
    }
  }, [])

  useEffect(() => {
    load(1)
  }, [load])

  const openCreate = () => {
    setForm({ username: '', email: '', role: 'user', password: '' })
    setDialogOpen(true)
  }

  const openEdit = (u: User) => {
    setForm({ id: u.id, username: u.username, email: u.email, role: u.role, password: '' })
    setDialogOpen(true)
  }

  const onSave = async () => {
    if (!form.username.trim()) return toast.warning('请输入用户名')
    if (!form.id && !form.password) return toast.warning('请输入密码')
    setSaving(true)
    try {
      if (form.id) {
        const payload: Record<string, unknown> = { username: form.username, email: form.email, role: form.role }
        if (form.password) payload.password = form.password
        await usersApi.update(form.id, payload)
      } else {
        await usersApi.create({
          username: form.username,
          email: form.email,
          role: form.role,
          password: form.password
        })
      }
      toast.success('保存成功')
      setDialogOpen(false)
      load(page)
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async () => {
    if (!deleteTarget) return
    try {
      await usersApi.remove(deleteTarget.id)
      toast.success('删除成功')
      setDeleteTarget(null)
      load(page)
    } catch (err) {
      toast.error((err as Error).message)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / 20))

  return (
    <div>
      <PageHeader
        title="用户管理"
        description="管理平台用户与角色权限。"
        actions={
          <Button onClick={openCreate}>
            <Plus className="h-4 w-4" /> 新增用户
          </Button>
        }
      />

      <div className="overflow-hidden rounded-md border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-14">ID</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>邮箱</TableHead>
              <TableHead>角色</TableHead>
              <TableHead className="w-40">创建时间</TableHead>
              <TableHead className="w-28 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.map((u) => (
              <TableRow key={u.id}>
                <TableCell className="text-muted-foreground">{u.id}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-2.5">
                    <UserAvatar avatar={u.avatar} email={u.email} username={u.nickname || u.username} size="sm" />
                    <div>
                      <div className="text-sm font-medium">{u.nickname || u.username}</div>
                      <div className="text-xs text-muted-foreground">@{u.username}</div>
                    </div>
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground">{u.email || '-'}</TableCell>
                <TableCell>
                  <Badge variant={u.role === 'admin' ? 'destructive' : 'secondary'} className="font-normal">
                    {u.role === 'admin' ? '管理员' : '普通用户'}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">{u.created_at}</TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <Button variant="ghost" size="sm" onClick={() => openEdit(u)}>
                      编辑
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      disabled={u.id === 1}
                      onClick={() => setDeleteTarget(u)}
                    >
                      删除
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
        <span>共 {total} 个用户</span>
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

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{form.id ? '编辑用户' : '新增用户'}</DialogTitle>
            <DialogDescription>{form.id ? '修改用户信息，密码留空则不修改。' : '创建一个新的平台用户。'}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="space-y-1.5">
              <Label>用户名 *</Label>
              <Input value={form.username} disabled={!!form.id} onChange={(e) => setForm({ ...form, username: e.target.value })} placeholder="登录用户名" />
            </div>
            <div className="space-y-1.5">
              <Label>邮箱</Label>
              <Input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} placeholder="邮箱（可选）" />
            </div>
            <div className="space-y-1.5">
              <Label>角色</Label>
              <Select value={form.role} onValueChange={(v) => setForm({ ...form, role: v })}>
                <SelectTrigger>
                  <SelectValue placeholder="选择角色" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="user">普通用户</SelectItem>
                  <SelectItem value="admin">管理员</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{form.id ? '重置密码' : '密码'} *</Label>
              <Input
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                placeholder={form.id ? '留空则不修改' : '请输入密码'}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={onSave} disabled={saving}>
              {saving ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={`确定删除用户「${deleteTarget?.username}」吗？`}
        description="删除后该用户将无法登录，其应用与绑定关系一并失效。"
        confirmText="删除"
        destructive
        onConfirm={onDelete}
      />
    </div>
  )
}
