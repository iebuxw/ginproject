<template>
  <div>
    <el-card>
      <div slot="header"><span>操作日志</span></div>
      <el-form :inline="true">
        <el-form-item>
          <el-select v-model="filters.method" placeholder="请求方式" clearable @change="fetchData">
            <el-option label="GET" value="GET"></el-option>
            <el-option label="POST" value="POST"></el-option>
            <el-option label="PUT" value="PUT"></el-option>
            <el-option label="DELETE" value="DELETE"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item><el-button @click="fetchData">查询</el-button></el-form-item>
        <el-form-item>
          <el-button type="primary" :disabled="exporting" @click="exportExcel">
            {{ exporting ? '导出中...' : '导出Excel' }}
          </el-button>
        </el-form-item>
      </el-form>
      <el-alert
        v-if="downloadUrl"
        title="导出完成"
        type="success"
        :closable="true"
        @close="downloadUrl = null"
        style="margin-bottom:15px"
      >
        <a href="#" @click.prevent="downloadFile">{{ downloadFilename || '点击下载' }}</a>
      </el-alert>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column prop="operator_id" label="操作人ID" width="80"></el-table-column>
        <el-table-column prop="method" label="方式" width="70"></el-table-column>
        <el-table-column prop="path" label="请求路径" width="160"></el-table-column>
        <el-table-column label="参数" min-width="200">
          <template slot-scope="{row}">
            <span class="params-preview">{{ row.params || '-' }}</span>
            <el-button v-if="row.params && row.params.length > 40" type="text" @click="showParams(row.params)">详情</el-button>
          </template>
        </el-table-column>
        <el-table-column prop="duration" label="耗时(ms)" width="80"></el-table-column>
        <el-table-column prop="ip" label="IP" width="140"></el-table-column>
        <el-table-column prop="created_at" label="操作时间" width="180"></el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <el-dialog title="请求参数" :visible.sync="dialogVisible" width="600px">
      <pre class="params-detail">{{ dialogContent }}</pre>
    </el-dialog>
  </div>
</template>
<script>
import axios from 'axios'
import { getLogs, exportLogs } from '@/api/log'
import { onWSMessage, offWSMessage } from '@/utils/ws'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0,
      filters: { method: '' },
      dialogVisible: false, dialogContent: '',
      exporting: false,
      currentTaskId: null,
      downloadUrl: null,
      downloadFilename: null,
    }
  },
  created() { this.fetchData() },
  mounted() {
    this._onComplete = this.onExportComplete.bind(this)
    this._onFailed = this.onExportFailed.bind(this)
    onWSMessage('export_complete', this._onComplete)
    onWSMessage('export_failed', this._onFailed)
  },
  beforeDestroy() {
    offWSMessage('export_complete', this._onComplete)
    offWSMessage('export_failed', this._onFailed)
  },
  methods: {
    async fetchData() {
      const res = await getLogs({ page: this.page, page_size: this.pageSize, method: this.filters.method })
      this.list = res.data.list; this.total = res.data.total
    },
    pageChange(p) { this.page = p; this.fetchData() },
    showParams(val) {
      this.dialogContent = val
      this.dialogVisible = true
    },
    exportExcel() {
      this.exporting = true
      this.downloadUrl = null
      exportLogs({ method: this.filters.method }).then(res => {
        this.currentTaskId = res.data.task_id
      }).catch(() => {
        this.exporting = false
        this.$message.error('导出请求失败')
      })
    },
    onExportComplete(msg) {
      if (msg.task_id !== this.currentTaskId) return
      this.exporting = false
      this.downloadUrl = msg.download_url
      this.downloadFilename = msg.filename
    },
    onExportFailed(msg) {
      if (msg.task_id !== this.currentTaskId) return
      this.exporting = false
      this.$message.error(msg.error || '导出失败')
    },
    downloadFile() {
      const token = this.$store.state.user.token
      axios.get(this.downloadUrl, {
        responseType: 'blob',
        headers: { Authorization: 'Bearer ' + token }
      }).then(res => {
        const url = window.URL.createObjectURL(new Blob([res.data]))
        const link = document.createElement('a')
        link.href = url
        link.download = this.downloadFilename || 'export.xlsx'
        document.body.appendChild(link)
        link.click()
        link.remove()
        window.URL.revokeObjectURL(url)
      }).catch(() => {
        this.$message.error('下载失败')
      })
    },
  }
}
</script>
<style scoped>
.params-preview {
  display: inline-block;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}
.params-detail {
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 400px;
  overflow-y: auto;
}
</style>
