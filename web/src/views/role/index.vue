<template>
  <div>
    <el-card>
      <div slot="header" class="clearfix">
        <span>角色管理</span>
        <el-button type="primary" size="small" style="float:right" @click="openDialog()">新增角色</el-button>
      </div>
      <el-input v-model="keyword" placeholder="搜索角色" style="width:250px;margin-bottom:10px" @keyup.enter.native="fetchData" clearable @clear="fetchData"></el-input>
      <el-button size="small" :loading="exporting" style="margin-left:10px" @click="exportExcel">{{ exporting ? '导出中...' : '导出Excel' }}</el-button>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column prop="name" label="角色名"></el-table-column>
        <el-table-column prop="code" label="角色标识"></el-table-column>
        <el-table-column prop="description" label="描述"></el-table-column>
        <el-table-column label="操作" width="180">
          <template slot-scope="s">
            <div style="white-space:nowrap">
              <el-button size="mini" @click="openDialog(s.row)">编辑</el-button>
              <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <el-dialog :title="isEdit ? '编辑角色' : '新增角色'" :visible.sync="dialogVisible" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="角色名"><el-input v-model="form.name"></el-input></el-form-item>
        <el-form-item label="角色标识"><el-input v-model="form.code"></el-input></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description"></el-input></el-form-item>
        <el-form-item label="菜单权限">
          <el-tree ref="menuTree" :data="allMenus" show-checkbox node-key="id" :default-checked-keys="form.menu_ids" :props="{ label: 'name', children: 'children' }"></el-tree>
        </el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="handleSubmit">确定</el-button></span>
    </el-dialog>
  </div>
</template>
<script>
import { getRoles, getRole, addRole, updateRole, deleteRole, exportRoles } from '@/api/role'
import { getMenus } from '@/api/menu'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0, keyword: '',
      exporting: false,
      dialogVisible: false, isEdit: false,
      form: { name: '', code: '', description: '', menu_ids: [] },
      allMenus: []
    }
  },
  created() { this.fetchData(); this.fetchMenus() },
  methods: {
    async fetchData() {
      const res = await getRoles({ page: this.page, page_size: this.pageSize, keyword: this.keyword })
      this.list = res.data.list; this.total = res.data.total
    },
    async fetchMenus() { const res = await getMenus(); this.allMenus = res.data },
    pageChange(p) { this.page = p; this.fetchData() },
    async openDialog(row) {
      if (row) {
        this.isEdit = true
        const res = await getRole(row.id)
        this.form = { ...res.data.role, menu_ids: res.data.menu_ids }
      } else {
        this.isEdit = false; this.form = { name: '', code: '', description: '', menu_ids: [] }
      }
      this.dialogVisible = true
      this.$nextTick(() => {
        if (this.$refs.menuTree) this.$refs.menuTree.setCheckedKeys(this.form.menu_ids)
      })
    },
    async handleSubmit() {
      const checked = this.$refs.menuTree.getCheckedKeys()
      const halfChecked = this.$refs.menuTree.getHalfCheckedKeys()
      const menu_ids = [...checked, ...halfChecked]
      const data = { ...this.form, menu_ids }
      if (this.isEdit) { await updateRole(this.form.id, data) }
      else { await addRole(data) }
      this.dialogVisible = false; this.fetchData()
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该角色?', '提示', { type: 'warning' })
      await deleteRole(id); this.fetchData(); this.$message.success('删除成功')
    },
    // 保存接口返回的文件流，优先从 content-disposition 解析文件名
    saveBlob(res, fallbackName) {
      const url = window.URL.createObjectURL(new Blob([res.data]))
      const link = document.createElement('a')
      link.href = url
      const disposition = res.headers['content-disposition'] || ''
      let filename = fallbackName
      const rfc5987 = disposition.match(/filename\*=UTF-8''(.+)/i)
      if (rfc5987) {
        filename = decodeURIComponent(rfc5987[1])
      } else {
        const fallback = disposition.match(/filename="?([^";]+)"?/)
        if (fallback) filename = fallback[1]
      }
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    },
    async exportExcel() {
      this.exporting = true
      try {
        const res = await exportRoles({ keyword: this.keyword })
        this.saveBlob(res, '角色列表.xlsx')
      } catch {
        this.$message.error('导出失败')
      } finally {
        this.exporting = false
      }
    }
  }
}
</script>
