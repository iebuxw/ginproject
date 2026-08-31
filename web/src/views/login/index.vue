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
        <el-form-item v-if="captchaEnabled" prop="captcha_code">
          <div style="display:flex;align-items:center">
            <el-input v-model="form.captcha_code" placeholder="验证码" style="flex:1" @keyup.enter.native="handleLogin"></el-input>
            <img v-if="captchaImage" :src="captchaImage" alt="验证码" style="height:40px;margin-left:10px;cursor:pointer;border-radius:4px" @click="refreshCaptcha" title="点击刷新" />
          </div>
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
import { getCaptcha } from '@/api/auth'
export default {
  data() {
    return {
      form: { username: '', password: '', captcha_id: '', captcha_code: '' },
      rules: {
        username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
        password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
      },
      loading: false,
      siteName: 'GinAdmin',
      captchaEnabled: false,
      captchaImage: ''
    }
  },
  created() {
    getSettings().then(res => {
      if (res.code === 200) {
        if (res.data.site_name) {
          this.siteName = res.data.site_name
          document.title = res.data.site_name
        }
        if (res.data.captcha_enabled === '1') {
          this.captchaEnabled = true
          this.refreshCaptcha()
        }
      }
    }).catch(() => {})
  },
  methods: {
    async refreshCaptcha() {
      try {
        const res = await getCaptcha()
        if (res.code === 200) {
          this.form.captcha_id = res.data.captcha_id
          this.captchaImage = res.data.captcha_image
          this.form.captcha_code = ''
        }
      } catch {}
    },
    async handleLogin() {
      const valid = await this.$refs.form.validate().catch(() => false)
      if (!valid) return
      this.loading = true
      try {
        await this.$store.dispatch('user/login', this.form)
        this.$router.push('/')
      } catch {
        if (this.captchaEnabled) {
          this.refreshCaptcha()
        }
        this.loading = false
      }
    }
  }
}
</script>
