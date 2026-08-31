<!-- web/src/views/log/setting.vue -->
<template>
  <div>
    <el-card>
      <div slot="header"><span>日志清理设置</span></div>
      <el-form ref="form" :model="form" label-width="100px" style="max-width:500px" v-loading="loading">
        <el-form-item label="保留天数" prop="days">
          <el-input-number v-model="form.days" :min="1" :max="3650" placeholder="保留天数"></el-input-number>
          <span style="margin-left:8px;color:#909399;font-size:12px">删除 N 天前的日志</span>
        </el-form-item>
        <el-form-item label="清理范围" prop="scope">
          <el-checkbox-group v-model="form.scope">
            <el-checkbox label="operation">操作日志</el-checkbox>
            <el-checkbox label="login">登录日志</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { getLogSettings, updateLogSettings } from '@/api/log-setting'

export default {
  data() {
    return {
      form: { days: 180, scope: ['operation', 'login'] },
      loading: false,
      saving: false
    }
  },
  created() {
    this.fetchSettings()
  },
  methods: {
    async fetchSettings() {
      this.loading = true
      try {
        const res = await getLogSettings()
        if (res.code === 200) {
          this.form.days = res.data.days || 180
          this.form.scope = res.data.scope || ['operation', 'login']
        }
      } finally {
        this.loading = false
      }
    },
    async handleSave() {
      if (this.form.scope.length === 0) {
        this.$message.warning('请至少选择一项清理范围')
        return
      }
      this.saving = true
      try {
        const res = await updateLogSettings({
          days: this.form.days,
          scope: this.form.scope
        })
        if (res.code === 200) {
          this.$message.success('保存成功')
        }
      } finally {
        this.saving = false
      }
    }
  }
}
</script>
