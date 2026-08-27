<template>
  <div>
    <el-alert
      v-if="backingUp"
      title="备份已在后台执行，请稍后刷新"
      type="success"
      :closable="false"
      show-icon
      style="margin-bottom:15px">
    </el-alert>

    <el-card>
      <div slot="header"><span>数据库备份</span></div>
      <el-form :inline="true">
        <el-form-item>
          <el-date-picker
            v-model="filters.dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="yyyy-MM-dd HH:mm:ss"
            @change="fetchData">
          </el-date-picker>
        </el-form-item>
        <el-form-item>
          <el-button @click="fetchData">查询</el-button>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="creating" @click="handleCreate">
            {{ creating ? '备份中...' : '新增备份' }}
          </el-button>
        </el-form-item>
      </el-form>

      <el-table :data="list" border v-loading="loading">
        <el-table-column prop="id" label="ID" width="80"></el-table-column>
        <el-table-column prop="filename" label="文件名" min-width="200"></el-table-column>
        <el-table-column label="文件大小" width="120">
          <template slot-scope="{row}">
            {{ formatSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column label="触发方式" width="100">
          <template slot-scope="{row}">
            <el-tag :type="row.trigger_type === 'cron' ? 'info' : ''">
              {{ row.trigger_type === 'cron' ? '定时' : '手动' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="100">
          <template slot-scope="{row}">
            <el-tag :type="row.type === 'snapshot' ? 'warning' : 'success'" size="small">
              {{ row.type === 'snapshot' ? '恢复快照' : '常规备份' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="{row}">
            <el-tag v-if="row.status === -1" type="warning">备份中...</el-tag>
            <el-tag v-else-if="row.status === 0" type="success">成功</el-tag>
            <el-tag v-else type="danger">失败</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="150"></el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180"></el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template slot-scope="{row}">
            <template v-if="row.status !== -1">
              <el-button type="text" size="small" @click="handleRestore(row)">恢复</el-button>
              <el-button type="text" size="small" @click="handleDownload(row)">下载</el-button>
              <el-button type="text" size="small" style="color:#F56C6C" @click="handleDelete(row)">删除</el-button>
            </template>
            <span v-else style="color:#909399">-</span>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <!-- 恢复确认对话框 -->
    <el-dialog title="确认恢复" :visible.sync="restoreDialogVisible" width="500px">
      <el-alert
        title="恢复操作将用备份文件覆盖当前数据库，此操作不可撤销。"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom:20px">
      </el-alert>
      <el-form>
        <el-form-item label="请输入 确认恢复 以继续：">
          <el-input v-model="restoreConfirmText" placeholder="确认恢复"></el-input>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="restoreDialogVisible = false">取消</el-button>
        <el-button type="danger" :disabled="restoreConfirmText !== '确认恢复'" :loading="restoring" @click="confirmRestore">
          确认恢复
        </el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { getBackups, createBackup, restoreBackup, deleteBackup } from '@/api/backup'
import axios from 'axios'

export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0,
      filters: { dateRange: [] },
      creating: false,
      loading: false,
      backingUp: false,
      pollingTimer: null,
      restoreDialogVisible: false,
      restoreConfirmText: '',
      restoring: false,
      currentRestoreRow: null,
    }
  },
  created() { this.fetchData() },
  beforeDestroy() {
    this.stopPolling()
  },
  methods: {
    async fetchData() {
      const params = { page: this.page, page_size: this.pageSize }
      if (this.filters.dateRange && this.filters.dateRange.length === 2) {
        params.start_time = this.filters.dateRange[0]
        params.end_time = this.filters.dateRange[1]
      }
      const res = await getBackups(params)
      this.list = res.data.list
      this.total = res.data.total

      // 检测备份是否完成（轮询时）
      if (this.backingUp) {
        const hasPending = this.list.some(r => r.status === -1)
        if (!hasPending) {
          this.stopPolling()
          this.$message.success('备份已完成')
        }
      }
    },
    pageChange(p) { this.page = p; this.fetchData() },
    formatSize(bytes) {
      if (!bytes || isNaN(bytes)) return '-'
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },
    async handleCreate() {
      this.creating = true
      try {
        await createBackup()
        // 在列表顶部插入临时记录
        this.list.unshift({
          id: '-',
          filename: '-',
          file_size: '-',
          trigger_type: 'manual',
          status: -1,
          type: 'backup',
          remark: '备份中...',
          created_at: new Date().toLocaleString()
        })
        this.total += 1
        this.backingUp = true
        this.startPolling()
        this.$message.success('备份任务已提交')
      } catch (e) {
        this.$message.error('备份失败')
      }
      this.creating = false
    },
    startPolling() {
      this.stopPolling()
      this.pollingTimer = setInterval(() => {
        this.fetchData()
      }, 3000)
    },
    stopPolling() {
      if (this.pollingTimer) {
        clearInterval(this.pollingTimer)
        this.pollingTimer = null
      }
      this.backingUp = false
    },
    handleRestore(row) {
      this.currentRestoreRow = row
      this.restoreConfirmText = ''
      this.restoreDialogVisible = true
    },
    async confirmRestore() {
      this.restoring = true
      try {
        await restoreBackup(this.currentRestoreRow.id)
        this.$message.success('恢复成功')
        this.restoreDialogVisible = false
      } catch (e) {
        this.$message.error('恢复失败')
      }
      this.restoring = false
    },
    handleDownload(row) {
      const token = this.$store.state.user.token
      axios.get(`/api/db-backups/${row.id}/download`, {
        responseType: 'blob',
        headers: { Authorization: 'Bearer ' + token }
      }).then(res => {
        const url = window.URL.createObjectURL(new Blob([res.data]))
        const link = document.createElement('a')
        link.href = url
        link.download = row.filename
        document.body.appendChild(link)
        link.click()
        link.remove()
        window.URL.revokeObjectURL(url)
      }).catch(() => {
        this.$message.error('下载失败')
      })
    },
    handleDelete(row) {
      this.$confirm('确定删除该备份记录？', '提示', {
        type: 'warning'
      }).then(async () => {
        await deleteBackup(row.id)
        this.$message.success('删除成功')
        this.fetchData()
      }).catch(() => {})
    },
  }
}
</script>
