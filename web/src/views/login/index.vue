<template>
  <div class="login-container" style="display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5">
    <el-card style="width:400px">
      <h2 style="text-align:center;margin-bottom:20px">{{ siteName }}</h2>
      <el-form ref="form" :model="form" :rules="rules">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="el-icon-user"></el-input>
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="el-icon-lock" @keyup.enter.native="handleLogin"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" style="width:100%" @click="handleLogin" :loading="loading">登录</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>
<script>
import { getSettings } from '@/api/setting'
export default {
  data() {
    return {
      form: { username: '', password: '' },
      rules: {
        username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
        password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
      },
      loading: false,
      siteName: 'GinAdmin'
    }
  },
  created() {
    getSettings().then(res => {
      if (res.code === 200 && res.data.site_name) {
        this.siteName = res.data.site_name
        document.title = res.data.site_name
      }
    }).catch(() => {})
  },
  methods: {
    async handleLogin() {
      const valid = await this.$refs.form.validate().catch(() => false)
      if (!valid) return
      this.loading = true
      try {
        await this.$store.dispatch('user/login', this.form)
        this.$router.push('/')
      } catch {
        this.loading = false
      }
    }
  }
}
</script>
