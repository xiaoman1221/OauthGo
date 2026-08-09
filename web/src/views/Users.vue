<template>
  <el-card>
    <div class="toolbar">
      <el-button type="primary" @click="openDialog()">新增用户</el-button>
    </div>

    <el-table :data="users" border stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="username" label="用户名" min-width="140" />
      <el-table-column prop="email" label="邮箱" min-width="180" />
      <el-table-column prop="role" label="角色" width="100">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'">
            {{ row.role === 'admin' ? '管理员' : '普通用户' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="170" />
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openDialog(row)">编辑</el-button>
          <el-button size="small" type="danger" :disabled="row.id === 1" @click="onDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑用户' : '新增用户'" width="440px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="用户名" required>
          <el-input v-model="form.username" :disabled="!!form.id" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.role">
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
        <el-form-item :label="form.id ? '重置密码' : '密码'" required>
          <el-input v-model="form.password" type="password" show-password
            :placeholder="form.id ? '留空则不修改' : '请输入密码'" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listUsers, createUser, updateUser, deleteUser } from '../api/modules'

const users = ref([])
const dialogVisible = ref(false)
const saving = ref(false)
const form = ref({})

onMounted(load)

async function load() {
  const data = await listUsers({ page: 1, page_size: 100 })
  users.value = data.list
}

function openDialog(row) {
  form.value = row
    ? { id: row.id, username: row.username, email: row.email, role: row.role, password: '' }
    : { username: '', email: '', role: 'user', password: '' }
  dialogVisible.value = true
}

async function onSave() {
  if (form.value.id) {
    const payload = { ...form.value }
    if (!payload.password) delete payload.password
    await updateUser(form.value.id, payload)
  } else {
    await createUser({ ...form.value })
  }
  ElMessage.success('保存成功')
  dialogVisible.value = false
  load()
}

async function onDelete(row) {
  await ElMessageBox.confirm(`确定删除用户「${row.username}」吗？`, '提示', { type: 'warning' })
  await deleteUser(row.id)
  ElMessage.success('删除成功')
  load()
}
</script>

<style scoped>
.toolbar {
  margin-bottom: 16px;
}
</style>
