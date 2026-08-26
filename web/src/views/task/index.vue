<template>
  <div>
    <el-card>
      <div slot="header" class="clearfix">
        <span>定时任务</span>
        <el-button type="primary" size="small" style="float:right" @click="openDialog()">新建任务</el-button>
      </div>
      <el-input v-model="keyword" placeholder="搜索任务名称/命令" style="width:250px;margin-bottom:10px" @keyup.enter.native="fetchData" clearable @clear="fetchData"></el-input>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column prop="name" label="任务名称"></el-table-column>
        <el-table-column prop="command" label="命令" width="140">
          <template slot-scope="s">
            <span v-if="s.row.command">{{ commandLabel(s.row.command) }}</span>
            <span v-else style="color:#909399">自定义</span>
          </template>
        </el-table-column>
        <el-table-column prop="cron" label="Cron 表达式" width="160"></el-table-column>
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
        <el-table-column label="操作" width="220">
          <template slot-scope="s">
            <el-button size="mini" @click="openDialog(s.row)">编辑</el-button>
            <el-button size="mini" type="primary" @click="handleRun(s.row.id)">立即执行</el-button>
            <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>

    <!-- 新建/编辑对话框 -->
    <el-dialog :title="isEdit ? '编辑任务' : '新建任务'" :visible.sync="dialogVisible" width="600px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="任务名称">
          <el-input v-model="form.name" placeholder="任务名称"></el-input>
        </el-form-item>
        <el-form-item label="命令">
          <el-select v-model="form.command" style="width:100%" @change="onCommandChange">
            <el-option v-for="c in commands" :key="c.name" :label="c.label" :value="c.name"></el-option>
            <el-option label="自定义" value="_custom"></el-option>
          </el-select>
        </el-form-item>
        <!-- 自定义模式字段 -->
        <template v-if="form.command === '_custom' || form.command === ''">
          <el-form-item label="回调地址">
            <el-input v-model="form.url" placeholder="http://example.com/callback"></el-input>
          </el-form-item>
          <el-form-item label="请求方式">
            <el-select v-model="form.method" style="width:100%">
              <el-option label="GET" value="GET"></el-option>
              <el-option label="POST" value="POST"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="请求头">
            <el-input v-model="form.headers" type="textarea" :rows="2" placeholder='JSON 对象，如 {"Content-Type": "application/json"}'></el-input>
          </el-form-item>
          <el-form-item v-if="form.method === 'POST'" label="请求体">
            <el-input v-model="form.body" type="textarea" :rows="3" placeholder="POST 请求体"></el-input>
          </el-form-item>
          <el-form-item label="超时（秒）">
            <el-input-number v-model="form.timeout" :min="1" :max="300"></el-input-number>
          </el-form-item>
        </template>
        <!-- Cron 表达式 + 快捷按钮 -->
        <el-form-item label="Cron 表达式">
          <el-input v-model="form.cron" placeholder="秒 分 时 日 月 周，如 0 0/5 * * * ?"></el-input>
        </el-form-item>
        <el-form-item label="快捷设置">
          <el-button-group>
            <el-button size="small" @click="setCron('minute')">每分钟</el-button>
            <el-button size="small" @click="setCron('hourly')">每小时</el-button>
            <el-button size="small" @click="setCron('daily')">每天</el-button>
            <el-button size="small" @click="setCron('weekly')">每周</el-button>
            <el-button size="small" @click="setCron('monthly')">每月</el-button>
          </el-button-group>
          <span style="margin-left:10px;color:#909399;font-size:12px">{{ cronHint }}</span>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark"></el-input>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { getCronTasks, addCronTask, updateCronTask, deleteCronTask, updateCronTaskStatus, runCronTask, getCronCommands } from '@/api/cron'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0, keyword: '',
      dialogVisible: false, isEdit: false,
      form: { name: '', command: '', url: '', method: 'GET', headers: '', body: '', cron: '', timeout: 30, remark: '' },
      commands: [], cronHour: 3
    }
  },
  computed: {
    commandLabel() {
      return (name) => {
        const c = this.commands.find(x => x.name === name)
        return c ? c.label : name
      }
    },
    cronHint() {
      const c = this.form.cron
      if (!c) return ''
      const parts = c.split(' ')
      if (parts.length < 6) return ''
      const h = parts[2], m = parts[1], dom = parts[3], mon = parts[4], dow = parts[5]
      if (m === '*' && h === '*') return '每分钟执行'
      if (m === '0' && h === '*') return '每小时整点执行'
      if (dom === '*' && mon === '*' && dow === '*') return `每天 ${h.padStart(2, '0')}:${m.padStart(2, '0')} 执行`
      if (dom === '?' && mon === '*' && dow !== '*') {
        const dayMap = { '1': '一', '2': '二', '3': '三', '4': '四', '5': '五', '6': '六', '0': '日' }
        return `每周${dayMap[dow] || dow} ${h.padStart(2, '0')}:${m.padStart(2, '0')} 执行`
      }
      if (dom !== '*' && mon === '*' && dow === '?') return `每月 ${dom} 号 ${h.padStart(2, '0')}:${m.padStart(2, '0')} 执行`
      return ''
    }
  },
  created() {
    this.fetchData()
    this.fetchCommands()
  },
  methods: {
    async fetchData() {
      const res = await getCronTasks({ page: this.page, page_size: this.pageSize, keyword: this.keyword })
      this.list = res.data.list; this.total = res.data.total
    },
    async fetchCommands() {
      const res = await getCronCommands()
      this.commands = res.data
    },
    pageChange(p) { this.page = p; this.fetchData() },
    openDialog(row) {
      if (row) {
        this.isEdit = true
        this.form = { ...row, command: row.command || '_custom' }
      } else {
        this.isEdit = false
        this.form = { name: '', command: '', url: '', method: 'GET', headers: '', body: '', cron: '', timeout: 30, remark: '' }
      }
      this.dialogVisible = true
    },
    onCommandChange(val) {
      if (val === '_custom' || val === '') {
        // 自定义模式：保留现有字段
      } else {
        // 预定义命令：清空 HTTP 字段
        this.form.url = ''
        this.form.method = 'GET'
        this.form.headers = ''
        this.form.body = ''
      }
    },
    setCron(type) {
      const h = String(this.cronHour).padStart(2, '0')
      const map = {
        minute: '0 * * * * *',
        hourly: `0 0 * * * *`,
        daily: `0 0 ${h} * * *`,
        weekly: `0 0 ${h} ? * 1`,
        monthly: `0 0 ${h} 1 * *`
      }
      this.form.cron = map[type]
    },
    handleStatusChange(row, val) {
      updateCronTaskStatus(row.id, { status: val }).catch(() => {
        this.$message.error('状态切换失败')
        row.status = val === 1 ? 0 : 1
      })
    },
    async handleSubmit() {
      if (!this.form.name) { this.$message.warning('任务名称不能为空'); return }
      if (!this.form.command && !this.form.url) { this.$message.warning('回调地址不能为空'); return }
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
    }
  }
}
</script>
