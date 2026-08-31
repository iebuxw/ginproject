<template>
  <div>
    <el-card>
      <div slot="header"><span>系统配置</span></div>
      <el-form ref="form" :model="form" label-width="80px" style="max-width:500px" v-loading="loading">
        <el-form-item label="站点名称" prop="site_name" :rules="[{ required: true, message: '请输入站点名称', trigger: 'blur' }]">
          <el-input v-model="form.site_name" maxlength="50" show-word-limit placeholder="请输入站点名称"></el-input>
        </el-form-item>
        <el-form-item label="Logo">
          <div style="display:flex;align-items:center">
            <el-image
              v-if="form.site_logo"
              :src="form.site_logo"
              :preview-src-list="[form.site_logo]"
              style="width:48px;height:48px;margin-right:12px;border-radius:4px"
              fit="contain"
            ></el-image>
            <el-upload
              action="/api/upload/logo"
              :headers="uploadHeaders"
              :show-file-list="false"
              :on-success="handleLogoSuccess"
              :before-upload="beforeLogoUpload"
            >
              <el-button size="small" type="primary">上传 Logo</el-button>
            </el-upload>
            <el-button
              v-if="form.site_logo"
              size="small"
              type="danger"
              icon="el-icon-delete"
              circle
              style="margin-left:8px"
              @click="handleRemoveLogo"
            ></el-button>
          </div>
        </el-form-item>
        <el-form-item label="登录验证码">
          <el-switch
            v-model="form.captcha_enabled"
            active-value="1"
            inactive-value="0"
            active-text="启用"
            inactive-text="禁用"
          ></el-switch>
          <div style="color:#999;font-size:12px;margin-top:4px">启用后登录时需要输入图片验证码</div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { mapState } from 'vuex'
import store from '@/store'
import { getSettings, updateSettings } from '@/api/setting'

export default {
  data() {
    return {
      form: { site_name: '', site_logo: '', captcha_enabled: '0' },
      loading: false,
      saving: false
    }
  },
  computed: {
    ...mapState('user', ['userInfo']),
    uploadHeaders() {
      return { Authorization: 'Bearer ' + store.state.user.token }
    }
  },
  created() {
    this.fetchSettings()
  },
  methods: {
    async fetchSettings() {
      this.loading = true
      try {
        const res = await getSettings()
        if (res.code === 200) {
          this.form = {
            site_name: res.data.site_name || '',
            site_logo: res.data.site_logo || '',
            captcha_enabled: res.data.captcha_enabled || '0'
          }
        }
      } finally {
        this.loading = false
      }
    },
    handleLogoSuccess(res) {
      if (res.code === 200) {
        this.form.site_logo = res.data.url
        this.$message.success('Logo 上传成功')
      }
    },
    beforeLogoUpload(file) {
      const isImage = ['image/jpeg', 'image/png', 'image/gif'].includes(file.type)
      const isLt2M = file.size / 1024 / 1024 < 2
      if (!isImage) { this.$message.error('只能上传 jpg/png/gif 格式') }
      if (!isLt2M) { this.$message.error('图片大小不能超过 2MB') }
      return isImage && isLt2M
    },
    handleRemoveLogo() {
      this.form.site_logo = ''
    },
    handleSave() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) return
        this.saving = true
        try {
          const res = await updateSettings(this.form)
          if (res.code === 200) {
            this.$message.success('保存成功')
            store.commit('settings/SET_SETTINGS', {
              siteName: this.form.site_name,
              siteLogo: this.form.site_logo
            })
            document.title = this.form.site_name || 'GinAdmin'
          }
        } finally {
          this.saving = false
        }
      })
    }
  }
}
</script>
