<template>
  <div>
    <el-card>
      <div slot="header" class="clearfix">
        <span>用户管理</span>
        <el-button type="primary" size="small" style="float:right" @click="openDialog()">新增用户</el-button>
      </div>
      <el-input v-model="keyword" placeholder="搜索用户名/邮箱" style="width:250px;margin-bottom:10px" @keyup.enter.native="fetchData" clearable @clear="fetchData"></el-input>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column prop="username" label="用户名"></el-table-column>
        <el-table-column prop="email" label="邮箱"></el-table-column>
        <el-table-column prop="phone" label="手机号"></el-table-column>
        <el-table-column prop="description" label="描述" show-overflow-tooltip></el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="s">
            <el-tag :type="s.row.status === 1 ? 'success' : 'danger'">{{ s.row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="角色">
          <template slot-scope="s">{{ (s.row.roles || []).map(r => r.name).join(',') }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template slot-scope="s">
            <el-button size="mini" @click="openDialog(s.row)">编辑</el-button>
            <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <el-dialog :title="isEdit ? '编辑用户' : '新增用户'" :visible.sync="dialogVisible" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="用户名"><el-input v-model="form.username"></el-input></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" :placeholder="isEdit ? '留空不修改' : ''"></el-input></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="form.email"></el-input></el-form-item>
        <el-form-item label="手机号"><el-input v-model="form.phone"></el-input></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description"></el-input></el-form-item>
        <el-form-item label="状态"><el-switch v-model="form.status" :active-value="1" :inactive-value="0"></el-switch></el-form-item>
        <el-form-item label="角色">
          <el-checkbox-group v-model="form.role_ids">
            <el-checkbox v-for="r in allRoles" :key="r.id" :label="r.id">{{ r.name }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="handleSubmit">确定</el-button></span>
    </el-dialog>
  </div>
</template>
<script>
import { getUsers, addUser, updateUser, deleteUser } from '@/api/user'
import { getRoles } from '@/api/role'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0, keyword: '',
      dialogVisible: false, isEdit: false,
      form: { username: '', password: '', email: '', phone: '', description: '', status: 1, role_ids: [] },
      allRoles: []
    }
  },
  created() { this.fetchData(); this.fetchRoles() },
  methods: {
    async fetchData() {
      const res = await getUsers({ page: this.page, page_size: this.pageSize, keyword: this.keyword })
      this.list = res.data.list; this.total = res.data.total
    },
    async fetchRoles() {
      const res = await getRoles({ page_size: 100 })
      this.allRoles = res.data.list
    },
    pageChange(p) { this.page = p; this.fetchData() },
    openDialog(row) {
      if (row) {
        this.isEdit = true
        this.form = { ...row, password: '', role_ids: (row.roles || []).map(r => r.id) }
      } else {
        this.isEdit = false
        this.form = { username: '', password: '', email: '', phone: '', description: '', status: 1, role_ids: [] }
      }
      this.dialogVisible = true
    },
    async handleSubmit() {
      if (this.isEdit) { await updateUser(this.form.id, this.form) }
      else { await addUser(this.form) }
      this.dialogVisible = false; this.fetchData()
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该用户?', '提示', { type: 'warning' })
      await deleteUser(id); this.fetchData(); this.$message.success('删除成功')
    }
  }
}
</script>
