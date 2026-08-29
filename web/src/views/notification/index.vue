<template>
  <div class="notification-container">
    <el-card shadow="never">
      <div slot="header" class="clearfix">
        <span>消息中心</span>
        <el-button size="mini" style="float:right" @click="handleReadAll">全部已读</el-button>
      </div>
      <div style="margin-bottom:10px">
        <el-radio-group v-model="activeType" size="small" @change="handleFilter">
          <el-radio-button :label="0">全部</el-radio-button>
          <el-radio-button :label="1">公告</el-radio-button>
          <el-radio-button :label="2">站内信</el-radio-button>
          <el-radio-button :label="3">系统事件</el-radio-button>
        </el-radio-group>
        <el-radio-group v-model="readStatus" size="small" style="margin-left:15px" @change="handleFilter">
          <el-radio-button :label="0">全部</el-radio-button>
          <el-radio-button :label="1">未读</el-radio-button>
          <el-radio-button :label="2">已读</el-radio-button>
        </el-radio-group>
      </div>
      <el-table :data="list" border v-loading="loading">
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="s">
            <el-tag :type="typeTag(s.row.type)" size="mini">{{ typeText(s.row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" show-overflow-tooltip>
          <template slot-scope="s">
            <span :style="s.row.read_at ? '' : 'font-weight:bold'">{{ s.row.title }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="发布时间" width="160" align="center"></el-table-column>
        <el-table-column label="状态" width="70" align="center">
          <template slot-scope="s">
            <el-tag :type="s.row.read_at ? 'info' : 'danger'" size="mini">{{ s.row.read_at ? '已读' : '未读' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center">
          <template slot-scope="s">
            <el-button size="mini" type="primary" plain @click="openDetail(s.row)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination small @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total, prev, pager, next" style="margin-top:10px;text-align:right"></el-pagination>
    </el-card>

    <!-- 消息详情 -->
    <el-dialog :title="detail ? detail.title : ''" :visible.sync="detailVisible" width="500px">
      <div style="white-space:pre-wrap;line-height:1.8">{{ detail ? detail.content : '' }}</div>
      <span slot="footer">
        <el-button type="primary" @click="detailVisible = false">关闭</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { getMyNotifications, markRead } from '@/api/notification'

export default {
  name: 'NotificationCenter',
  data() {
    return {
      list: [],
      page: 1,
      pageSize: 10,
      total: 0,
      activeType: 0,
      readStatus: 0,
      loading: false,
      detail: null,
      detailVisible: false
    }
  },
  created() {
    this.fetchList()
  },
  methods: {
    typeText(t) {
      return { 1: '公告', 2: '站内信', 3: '系统事件' }[t] || '未知'
    },
    typeTag(t) {
      return { 1: 'success', 2: '', 3: 'warning' }[t] || 'info'
    },
    handleFilter() {
      this.page = 1
      this.fetchList()
    },
    pageChange(p) {
      this.page = p
      this.fetchList()
    },
    async fetchList() {
      this.loading = true
      try {
        const res = await getMyNotifications({
          page: this.page,
          page_size: this.pageSize,
          read_status: this.readStatus,
          type: this.activeType
        })
        if (res.code === 200) {
          this.list = res.data.list || []
          this.total = res.data.total || 0
        }
      } finally {
        this.loading = false
      }
    },
    async openDetail(row) {
      this.detail = row
      this.detailVisible = true
      if (!row.read_at) {
        try {
          await markRead({ ids: [row.id] })
          row.read_at = 'just-now'
        } catch (e) { /* 已读失败不阻塞阅读 */ }
      }
    },
    async handleReadAll() {
      try {
        await markRead({ all: true })
        this.$message.success('已全部标记为已读')
        this.fetchList()
      } catch (e) { /* request.js 已统一提示 */ }
    }
  }
}
</script>

<style scoped>
.notification-container { padding: 0; }
</style>
