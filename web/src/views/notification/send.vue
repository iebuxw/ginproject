<template>
  <div class="notification-send-container">
    <el-card shadow="never">
      <div slot="header"><span>消息发送</span></div>
      <el-form :model="form" :rules="rules" ref="sendForm" label-width="90px" style="max-width:600px">
        <el-form-item label="消息类型" prop="type">
          <el-radio-group v-model="form.type">
            <el-radio :label="1">公告</el-radio>
            <el-radio :label="2">站内信</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" maxlength="200" show-word-limit placeholder="请输入标题"></el-input>
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input v-model="form.content" type="textarea" :rows="6" placeholder="请输入内容"></el-input>
        </el-form-item>
        <el-form-item label="接收范围" prop="target_type">
          <el-radio-group v-model="form.target_type">
            <el-radio :label="1">全员</el-radio>
            <el-radio :label="2">按角色</el-radio>
            <el-radio :label="3">指定用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.target_type === 2" label="收件角色" prop="role_ids">
          <el-select v-model="form.role_ids" multiple filterable placeholder="请选择角色" style="width:100%" @change="() => $refs.sendForm.validateField('target_type')">
            <el-option v-for="r in roles" :key="r.id" :label="r.name" :value="r.id"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.target_type === 3" label="收件用户" prop="user_ids">
          <el-select v-model="form.user_ids" multiple filterable placeholder="请选择用户" style="width:100%" @change="() => $refs.sendForm.validateField('target_type')">
            <el-option v-for="u in users" :key="u.id" :label="u.username" :value="u.id"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="sending" @click="handleSubmit">发 送</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { sendNotification } from '@/api/notification'
import { getRoles } from '@/api/role'
import { getUsers } from '@/api/user'

export default {
  name: 'NotificationSend',
  data() {
    const checkTargets = (rule, value, callback) => {
      if (this.form.target_type === 2 && this.form.role_ids.length === 0) {
        callback(new Error('请选择收件角色'))
      } else if (this.form.target_type === 3 && this.form.user_ids.length === 0) {
        callback(new Error('请选择收件用户'))
      } else {
        callback()
      }
    }
    return {
      form: {
        type: 1,
        title: '',
        content: '',
        target_type: 1,
        role_ids: [],
        user_ids: []
      },
      rules: {
        title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
        target_type: [{ validator: checkTargets, trigger: 'change' }]
      },
      roles: [],
      users: [],
      sending: false
    }
  },
  created() {
    this.fetchRoles()
    this.fetchUsers()
  },
  methods: {
    async fetchRoles() {
      try {
        const res = await getRoles({ page: 1, page_size: 100 })
        if (res.code === 200) this.roles = res.data.list || []
      } catch (e) { /* 静默 */ }
    },
    async fetchUsers() {
      try {
        const res = await getUsers({ page: 1, page_size: 200 })
        if (res.code === 200) this.users = res.data.list || []
      } catch (e) { /* 静默 */ }
    },
    async handleSubmit() {
      this.$refs.sendForm.validate(async valid => {
        if (!valid) return
        this.sending = true
        try {
          const res = await sendNotification(this.form)
          if (res.code === 200) {
            this.$message.success('发送成功')
            this.$refs.sendForm.resetFields()
            this.form.role_ids = []
            this.form.user_ids = []
          }
        } finally {
          this.sending = false
        }
      })
    }
  }
}
</script>
