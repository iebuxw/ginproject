<template>
  <div>
    <el-card>
      <div slot="header" class="clearfix">
        <span>定时任务</span>
        <el-button type="primary" size="small" style="float:right" @click="openDialog()">新建任务</el-button>
      </div>
      <el-input v-model="keyword" placeholder="搜索任务名称/URL" style="width:250px;margin-bottom:10px" @keyup.enter.native="fetchData" clearable @clear="fetchData"></el-input>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column prop="name" label="任务名称"></el-table-column>
        <el-table-column prop="url" label="回调地址" show-overflow-tooltip></el-table-column>
        <el-table-column label="方法" width="80">
          <template slot-scope="s">
            <el-tag :type="s.row.method === 'GET' ? 'success' : 'warning'">{{ s.row.method }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cron" label="cron 表达式" width="160"></el-table-column>
        <el-table-column label="状态" width="90">
          <template slot-scope="s">
            <el-switch v-model="s.row.status" :active-value="1" :inactive-value="0" @change="val => handleStatusChange(s.row, val)"></el-switch>
          </template>
        </el-table-column>
        <el-table-column label="最近执行" width="90">
          <template slot-scope="s">
            <el-tag v-if="s.row.last_exec_status === 0" type="success" size="mini">成功</el-tag>
            <el-tag v-else-if="s.row.last_exec_status === 1" type="danger" size="mini">失败</el-tag>
            <el-tag v-else-if="s.row.last_exec_status === 2" type="warning" size="mini">跳过</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260">
          <template slot-scope="s">
            <el-button size="mini" @click="openDialog(s.row)">编辑</el-button>
            <el-button size="mini" type="primary" @click="handleRun(s.row.id)">立即执行</el-button>
            <el-button size="mini" type="info" @click="openLog(s.row.id)">日志</el-button>
            <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <el-dialog :title="isEdit ? '编辑任务' : '新建任务'" :visible.sync="dialogVisible" width="600px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="任务名称"><el-input v-model="form.name" placeholder="任务名称"></el-input></el-form-item>
        <el-form-item label="回调地址"><el-input v-model="form.url" placeholder="http://example.com/callback"></el-input></el-form-item>
        <el-form-item label="请求方式">
          <el-select v-model="form.method" style="width:100%">
            <el-option label="GET" value="GET"></el-option>
            <el-option label="POST" value="POST"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="请求头"><el-input v-model="form.headers" type="textarea" :rows="2" placeholder='JSON 对象，如 {"Content-Type": "application/json"}'></el-input></el-form-item>
        <el-form-item v-if="form.method === 'POST'" label="请求体"><el-input v-model="form.body" type="textarea" :rows="3" placeholder="POST 请求体"></el-input></el-form-item>
        <el-form-item label="cron 表达式"><el-input v-model="form.cron" placeholder="秒 分 时 日 月 周，如 0 0/5 * * * ?"></el-input></el-form-item>
        <el-form-item label="超时（秒）"><el-input-number v-model="form.timeout" :min="1" :max="300"></el-input-number></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark"></el-input></el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="handleSubmit">确定</el-button></span>
    </el-dialog>

    <el-dialog title="执行日志" :visible.sync="logVisible" width="800px">
      <el-table :data="logList" border>
        <el-table-column label="触发方式" width="90">
          <template slot-scope="s">
            <el-tag :type="s.row.trigger === 'manual' ? 'warning' : 'primary'" size="mini">{{ s.row.trigger === 'manual' ? '手动' : '定时' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="s">
            <el-tag v-if="s.row.status === 0" type="success" size="mini">成功</el-tag>
            <el-tag v-else-if="s.row.status === 1" type="danger" size="mini">失败</el-tag>
            <el-tag v-else-if="s.row.status === 2" type="warning" size="mini">跳过</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="http_status" label="HTTP 状态" width="90"></el-table-column>
        <el-table-column prop="duration_ms" label="耗时(ms)" width="90"></el-table-column>
        <el-table-column prop="error_msg" label="错误信息" show-overflow-tooltip></el-table-column>
        <el-table-column prop="created_at" label="执行时间" width="170"></el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="logPageChange" :current-page="logPage" :page-size="logPageSize" :total="logTotal" layout="total,prev,pager,next"></el-pagination>
    </el-dialog>
  </div>
</template>
<script>
import { getCronTasks, addCronTask, updateCronTask, deleteCronTask, updateCronTaskStatus, runCronTask, getCronTaskExecutions } from '@/api/cron'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0, keyword: '',
      dialogVisible: false, isEdit: false,
      form: { name: '', url: '', method: 'GET', headers: '', body: '', cron: '', timeout: 30, remark: '' },
      logVisible: false, logList: [], logPage: 1, logPageSize: 10, logTotal: 0, logTaskID: 0
    }
  },
  created() { this.fetchData() },
  methods: {
    async fetchData() {
      const res = await getCronTasks({ page: this.page, page_size: this.pageSize, keyword: this.keyword })
      this.list = res.data.list; this.total = res.data.total
    },
    pageChange(p) { this.page = p; this.fetchData() },
    openDialog(row) {
      if (row) {
        this.isEdit = true
        this.form = { ...row }
      } else {
        this.isEdit = false
        this.form = { name: '', url: '', method: 'GET', headers: '', body: '', cron: '', timeout: 30, remark: '' }
      }
      this.dialogVisible = true
    },
    handleStatusChange(row, val) {
      updateCronTaskStatus(row.id, { status: val }).catch(() => {
        this.$message.error('状态切换失败')
        row.status = val === 1 ? 0 : 1
      })
    },
    async handleSubmit() {
      if (!this.form.name) { this.$message.warning('任务名称不能为空'); return }
      if (!this.form.url) { this.$message.warning('回调地址不能为空'); return }
      if (!this.form.cron) { this.$message.warning('cron 表达式不能为空'); return }
      if (this.isEdit) { await updateCronTask(this.form.id, this.form) } else { await addCronTask(this.form) }
      this.dialogVisible = false; this.fetchData(); this.$message.success(this.isEdit ? '编辑成功' : '新增成功')
    },
    async handleRun(id) {
      await this.$confirm('确认立即执行该任务?', '提示', { type: 'warning' })
      await runCronTask(id); this.$message.success('已触发执行')
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该任务?', '提示', { type: 'warning' })
      await deleteCronTask(id); this.fetchData(); this.$message.success('删除成功')
    },
    async openLog(id) {
      this.logTaskID = id; this.logPage = 1; this.logVisible = true; this.fetchLogs()
    },
    async fetchLogs() {
      const res = await getCronTaskExecutions(this.logTaskID, { page: this.logPage, page_size: this.logPageSize })
      this.logList = res.data.list; this.logTotal = res.data.total
    },
    logPageChange(p) { this.logPage = p; this.fetchLogs() }
  }
}
</script>
