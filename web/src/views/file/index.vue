<template>
  <div>
    <el-card>
      <div slot="header"><span>文件管理</span></div>
      <el-form :inline="true" @submit.native.prevent>
        <el-form-item label="文件名">
          <el-input v-model="filters.name" placeholder="请输入文件名" clearable @keyup.enter.native="handleSearch"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
        <el-form-item style="float:right">
          <el-upload
            :show-file-list="false"
            :before-upload="beforeUpload"
            :http-request="handleUpload">
            <el-button type="primary" icon="el-icon-upload2" :loading="uploading">上传文件</el-button>
          </el-upload>
        </el-form-item>
      </el-form>

      <el-table :data="list" border v-loading="loading">
        <el-table-column prop="id" label="ID" width="80"></el-table-column>
        <el-table-column label="预览" width="100" align="center">
          <template slot-scope="{row}">
            <el-image
              v-if="isImage(row.ext)"
              :src="previewUrl(row)"
              :preview-src-list="[previewUrl(row)]"
              fit="cover"
              style="width:40px;height:40px">
            </el-image>
            <i v-else class="el-icon-document" style="font-size:26px;color:#909399"></i>
          </template>
        </el-table-column>
        <el-table-column prop="original_name" label="文件名" min-width="200"></el-table-column>
        <el-table-column label="大小" width="120">
          <template slot-scope="{row}">{{ formatSize(row.size) }}</template>
        </el-table-column>
        <el-table-column prop="ext" label="类型" width="90"></el-table-column>
        <el-table-column prop="uploader_name" label="上传者" width="110"></el-table-column>
        <el-table-column prop="created_at" label="上传时间" width="180"></el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template slot-scope="{row}">
            <el-button type="text" size="small" @click="handleDownload(row)">下载</el-button>
            <el-button type="text" size="small" style="color:#F56C6C" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>
  </div>
</template>

<script>
import { getFiles, uploadFile, deleteFile } from '@/api/file'
import axios from 'axios'

export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0,
      filters: { name: '' },
      loading: false,
      uploading: false,
    }
  },
  created() { this.fetchData() },
  methods: {
    async fetchData() {
      this.loading = true
      try {
        const params = { page: this.page, page_size: this.pageSize }
        if (this.filters.name) params.name = this.filters.name
        const res = await getFiles(params)
        this.list = res.data.list
        this.total = res.data.total
      } finally {
        this.loading = false
      }
    },
    handleSearch() { this.page = 1; this.fetchData() },
    handleReset() { this.filters.name = ''; this.handleSearch() },
    pageChange(p) { this.page = p; this.fetchData() },
    isImage(ext) {
      return ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'].includes((ext || '').toLowerCase())
    },
    previewUrl(row) {
      return '/api/uploads/files/' + row.stored_name
    },
    formatSize(bytes) {
      if (!bytes || isNaN(bytes)) return '-'
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },
    beforeUpload(file) {
      const forbidden = ['exe', 'dll', 'bat', 'cmd', 'com', 'msi', 'vbs', 'sh', 'reg']
      const parts = file.name.split('.')
      const ext = parts.length > 1 ? parts.pop().toLowerCase() : ''
      if (forbidden.includes(ext)) {
        this.$message.error('不允许上传该文件类型')
        return false
      }
      if (file.size > 100 * 1024 * 1024) {
        this.$message.error('文件大小不能超过 100MB')
        return false
      }
      return true
    },
    handleUpload({ file }) {
      this.uploading = true
      const formData = new FormData()
      formData.append('file', file)
      uploadFile(formData).then(() => {
        this.$message.success('上传成功')
        this.page = 1
        this.fetchData()
      }).catch(() => {
        this.$message.error('上传失败')
      }).finally(() => {
        this.uploading = false
      })
    },
    handleDownload(row) {
      const token = this.$store.state.user.token
      axios.get(`/api/files/${row.id}/download`, {
        responseType: 'blob',
        headers: { Authorization: 'Bearer ' + token }
      }).then(res => {
        if ((res.headers['content-type'] || '').includes('application/json')) {
          this.$message.error('下载失败：文件不存在或已被删除')
          return
        }
        const url = window.URL.createObjectURL(new Blob([res.data]))
        const link = document.createElement('a')
        link.href = url
        link.download = row.original_name
        document.body.appendChild(link)
        link.click()
        link.remove()
        window.URL.revokeObjectURL(url)
      }).catch(() => {
        this.$message.error('下载失败')
      })
    },
    handleDelete(row) {
      this.$confirm(`确定删除文件「${row.original_name}」？删除后不可恢复`, '提示', {
        type: 'warning'
      }).then(async () => {
        await deleteFile(row.id)
        this.$message.success('删除成功')
        this.fetchData()
      }).catch(() => {})
    },
  }
}
</script>
