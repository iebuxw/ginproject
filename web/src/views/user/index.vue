<template>
  <div>
    <el-card>
      <div slot="header" class="clearfix">
        <span>用户管理</span>
        <div style="float:right">
          <el-button size="small" @click="importDialogVisible = true">导入用户</el-button>
          <el-button size="small" :loading="exporting" @click="exportExcel">{{ exporting ? '导出中...' : '导出Excel' }}</el-button>
          <el-button type="primary" size="small" @click="openDialog()">新增用户</el-button>
        </div>
      </div>
      <el-form :inline="true">
        <el-form-item>
          <el-input v-model="keyword" placeholder="搜索用户名/邮箱" clearable @keyup.enter.native="fetchData" style="width:250px"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button @click="fetchData">查询</el-button>
        </el-form-item>
      </el-form>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column label="头像" width="80">
          <template slot-scope="s">
            <el-avatar :size="36" :src="s.row.avatar">{{ (s.row.username || '').charAt(0) }}</el-avatar>
          </template>
        </el-table-column>
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
            <div style="white-space:nowrap">
              <el-button size="mini" @click="openDialog(s.row)">编辑</el-button>
              <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <el-dialog :title="isEdit ? '编辑用户' : '新增用户'" :visible.sync="dialogVisible" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="头像">
          <el-upload
            action="/api/upload/avatar"
            :headers="uploadHeaders"
            :show-file-list="false"
            :on-success="handleAvatarSuccess"
            :before-upload="beforeAvatarUpload"
          >
            <el-avatar :size="60" :src="form.avatar">{{ (form.username || '').charAt(0) }}</el-avatar>
            <div style="margin-top:4px"><el-button size="mini" type="text">上传头像</el-button></div>
          </el-upload>
        </el-form-item>
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

    <el-dialog title="导入用户" :visible.sync="importDialogVisible" width="520px">
      <el-alert type="info" :closable="false" show-icon title="请先下载模板，按模板格式填写后上传导入"
        style="margin-bottom:15px"></el-alert>
      <el-upload drag action="" :auto-upload="false" :limit="1"
        :on-change="handleImportChange" :on-remove="handleImportRemove" :file-list="importFileList">
        <i class="el-icon-upload"></i>
        <div class="el-upload__text">将文件拖到此处，或<em>点击选择</em></div>
        <div class="el-upload__tip" slot="tip">仅支持 .xlsx 文件，<el-link type="primary" :underline="false" style="font-size:12px" @click="downloadTemplate">下载模板</el-link></div>
      </el-upload>
      <div v-if="importResult" style="margin-top:15px">
        <el-alert :type="importResult.failed.length ? 'warning' : 'success'" :closable="false" show-icon
          :title="'共 ' + importResult.total + ' 条：成功 ' + importResult.success + '，跳过 ' + importResult.skipped + '，失败 ' + importResult.failed.length"></el-alert>
        <div v-if="importResult.skipped_usernames.length" style="margin-top:10px;font-size:13px">
          跳过的用户名：{{ importResult.skipped_usernames.join('、') }}
        </div>
        <div v-if="importResult.failed.length" style="margin-top:10px">
          <div v-for="fitem in importResult.failed" :key="fitem.row" style="font-size:13px;color:#F56C6C">
            第 {{ fitem.row }} 行：{{ fitem.reason }}
          </div>
        </div>
      </div>
      <span slot="footer">
        <el-button @click="importDialogVisible = false">关闭</el-button>
        <el-button type="primary" :loading="importing" :disabled="!importFile" @click="handleImport">开始导入</el-button>
      </span>
    </el-dialog>
  </div>
</template>
<script>
import { getUsers, addUser, updateUser, deleteUser, exportUsers, importUsers, downloadImportTemplate } from '@/api/user'
import store from '@/store'
import { getRoles } from '@/api/role'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0, keyword: '', exporting: false,
      dialogVisible: false, isEdit: false,
      form: { username: '', password: '', email: '', phone: '', description: '', avatar: '', status: 1, role_ids: [] },
      allRoles: [],
      importDialogVisible: false, importing: false, importFile: null, importFileList: [], importResult: null
    }
  },
  created() { this.fetchData(); this.fetchRoles() },
  computed: {
    uploadHeaders() {
      return { Authorization: 'Bearer ' + store.state.user.token }
    }
  },
  methods: {
    handleAvatarSuccess(res) {
      if (res.code === 200) { this.form.avatar = res.data.url }
    },
    beforeAvatarUpload(file) {
      const isImage = ['image/jpeg', 'image/png', 'image/gif'].includes(file.type)
      const isLt2M = file.size / 1024 / 1024 < 2
      if (!isImage) { this.$message.error('只能上传 jpg/png/gif 格式') }
      if (!isLt2M) { this.$message.error('图片大小不能超过 2MB') }
      return isImage && isLt2M
    },
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
        this.form = { username: '', password: '', email: '', phone: '', description: '', avatar: '', status: 1, role_ids: [] }
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
    },
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
      link.remove()
      window.URL.revokeObjectURL(url)
    },
    async downloadTemplate() {
      const res = await downloadImportTemplate()
      this.saveBlob(res, '用户导入模板.xlsx')
    },
    handleImportChange(file, fileList) {
      this.importFile = file.raw
      this.importFileList = fileList.slice(-1)
    },
    handleImportRemove() {
      this.importFile = null
      this.importFileList = []
    },
    async handleImport() {
      this.importing = true
      this.importResult = null
      try {
        const fd = new FormData()
        fd.append('file', this.importFile)
        const res = await importUsers(fd)
        this.importResult = res.data
        this.fetchData()
      } finally {
        this.importing = false
      }
    },
    async exportExcel() {
      this.exporting = true
      try {
        const res = await exportUsers({ keyword: this.keyword })
        this.saveBlob(res, '用户列表.xlsx')
      } catch {
        this.$message.error('导出失败')
      } finally {
        this.exporting = false
      }
    }
  }
}
</script>
