<template>
  <div>
    <el-card>
      <div slot="header"><span>执行日志</span></div>
      <!-- 筛选区 -->
      <el-form :inline="true" style="margin-bottom:15px">
        <el-form-item label="任务名称">
          <el-select v-model="filter.taskId" placeholder="全部" clearable style="width:180px" @change="handleSearch">
            <el-option v-for="t in taskOptions" :key="t.id" :label="t.name" :value="t.id"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filter.status" placeholder="全部" clearable style="width:120px" @change="handleSearch">
            <el-option label="成功" :value="0"></el-option>
            <el-option label="失败" :value="1"></el-option>
            <el-option label="跳过" :value="2"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围">
          <el-date-picker v-model="filter.dateRange" type="daterange" range-separator="-" start-placeholder="开始日期" end-placeholder="结束日期" value-format="yyyy-MM-dd" style="width:260px" @change="handleSearch"></el-date-picker>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
      <!-- 表格 -->
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="70"></el-table-column>
        <el-table-column label="任务名称" width="160">
          <template slot-scope="s">
            {{ taskName(s.row.task_id) }}
          </template>
        </el-table-column>
        <el-table-column label="命令" width="140">
          <template slot-scope="s">
            {{ taskCommand(s.row.task_id) || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="s">
            <el-tag v-if="s.row.status === 0" type="success" size="mini">成功</el-tag>
            <el-tag v-else-if="s.row.status === 1" type="danger" size="mini">失败</el-tag>
            <el-tag v-else-if="s.row.status === 2" type="warning" size="mini">跳过</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时(秒)" width="100">
          <template slot-scope="s">
            {{ (s.row.duration_ms / 1000).toFixed(1) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="执行时间" width="170"></el-table-column>
        <el-table-column label="操作" width="150">
          <template slot-scope="s">
            <el-button size="mini" type="info" @click="viewOutput(s.row)">查看输出</el-button>
            <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <!-- 查看输出弹窗 -->
    <el-dialog title="执行输出" :visible.sync="outputVisible" width="700px">
      <pre style="max-height:400px;overflow:auto;background:#f5f7fa;padding:15px;border-radius:4px;font-size:13px;white-space:pre-wrap;word-break:break-all">{{ outputContent }}</pre>
      <span slot="footer"><el-button @click="outputVisible = false">关闭</el-button></span>
    </el-dialog>
  </div>
</template>

<script>
import { getAllCronTaskExecutions, getCronTasks } from '@/api/cron'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0,
      filter: { taskId: '', status: '', dateRange: null },
      taskOptions: [],
      outputVisible: false, outputContent: ''
    }
  },
  created() {
    this.fetchTasks()
    // 支持从任务列表页跳转过来带 task_id 参数
    if (this.$route.query.task_id) {
      this.filter.taskId = Number(this.$route.query.task_id)
    }
    this.fetchData()
  },
  methods: {
    async fetchTasks() {
      const res = await getCronTasks({ page: 1, page_size: 100 })
      this.taskOptions = res.data.list
    },
    async fetchData() {
      const params = { page: this.page, page_size: this.pageSize }
      if (this.filter.taskId) params.task_id = this.filter.taskId
      if (this.filter.status !== '' && this.filter.status !== null) params.status = this.filter.status
      if (this.filter.dateRange && this.filter.dateRange.length === 2) {
        params.start_time = this.filter.dateRange[0]
        params.end_time = this.filter.dateRange[1]
      }
      const res = await getAllCronTaskExecutions(params)
      this.list = res.data.list; this.total = res.data.total
    },
    taskName(taskId) {
      const t = this.taskOptions.find(x => x.id === taskId)
      return t ? t.name : taskId
    },
    taskCommand(taskId) {
      const t = this.taskOptions.find(x => x.id === taskId)
      return t ? t.command : ''
    },
    handleSearch() { this.page = 1; this.fetchData() },
    handleReset() { this.filter = { taskId: '', status: '', dateRange: null }; this.handleSearch() },
    pageChange(p) { this.page = p; this.fetchData() },
    viewOutput(row) {
      this.outputContent = row.response || row.error_msg || '(无输出)'
      this.outputVisible = true
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该日志?', '提示', { type: 'warning' })
      const { default: request } = await import('@/api/request')
      await request.delete('/cron-tasks/executions/' + id)
      this.fetchData(); this.$message.success('删除成功')
    }
  }
}
</script>
