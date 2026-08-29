<template>
  <div>
    <el-card style="margin-bottom:20px">
      <div slot="header"><span>基本信息</span></div>
      <el-form ref="profileFormRef" :model="profileForm" :rules="profileRules" label-width="80px" style="max-width:500px">
        <el-form-item label="用户名">
          <span>{{ userInfo.username }}</span>
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="profileForm.email" placeholder="请输入邮箱"></el-input>
        </el-form-item>
        <el-form-item label="头像">
          <div style="display:flex;align-items:center">
            <el-avatar :size="60" :src="userInfo.avatar" style="margin-right:16px">
              {{ (userInfo.username || '用户').charAt(0) }}
            </el-avatar>
            <el-upload
              action="/api/upload/avatar"
              :headers="uploadHeaders"
              :show-file-list="false"
              :on-success="handleAvatarSuccess"
              :before-upload="beforeAvatarUpload"
            >
              <el-button size="small" type="primary">更换头像</el-button>
            </el-upload>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleUpdateProfile">保存</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <div slot="header"><span>修改密码</span></div>
      <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="80px" style="max-width:500px">
        <el-form-item label="原密码" prop="old_password">
          <el-input v-model="pwdForm.old_password" type="password" placeholder="请输入原密码"></el-input>
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input v-model="pwdForm.new_password" type="password" placeholder="不少于6位"></el-input>
        </el-form-item>
        <el-form-item label="确认密码" prop="confirm_password">
          <el-input v-model="pwdForm.confirm_password" type="password" placeholder="再次输入新密码"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleChangePassword">确认修改</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { mapState } from 'vuex'
import store from '@/store'

export default {
  data() {
    const validateConfirm = (rule, value, callback) => {
      if (value !== this.pwdForm.new_password) {
        callback(new Error('两次输入的密码不一致'))
      } else {
        callback()
      }
    }
    return {
      profileForm: { email: '' },
      profileRules: {
        email: [
          { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
        ]
      },
      pwdForm: { old_password: '', new_password: '', confirm_password: '' },
      pwdRules: {
        old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
        new_password: [
          { required: true, message: '请输入新密码', trigger: 'blur' },
          { min: 6, message: '密码不少于6位', trigger: 'blur' }
        ],
        confirm_password: [
          { required: true, message: '请再次输入新密码', trigger: 'blur' },
          { validator: validateConfirm, trigger: 'blur' }
        ]
      }
    }
  },
  computed: {
    ...mapState('user', ['userInfo']),
    uploadHeaders() {
      return { Authorization: 'Bearer ' + store.state.user.token }
    }
  },
  watch: {
    userInfo: {
      handler(val) {
        if (val) {
          this.profileForm.email = val.email || ''
        }
      },
      immediate: true
    }
  },
  methods: {
    async handleAvatarSuccess(res) {
      if (res.code === 200) {
        await this.$store.dispatch('user/getUserInfo')
        this.$message.success('头像更新成功')
      }
    },
    beforeAvatarUpload(file) {
      const isImage = ['image/jpeg', 'image/png', 'image/gif'].includes(file.type)
      const isLt2M = file.size / 1024 / 1024 < 2
      if (!isImage) { this.$message.error('只能上传 jpg/png/gif 格式') }
      if (!isLt2M) { this.$message.error('图片大小不能超过 2MB') }
      return isImage && isLt2M
    },
    handleUpdateProfile() {
      this.$refs.profileFormRef.validate(async (valid) => {
        if (!valid) return
        try {
          await this.$store.dispatch('user/updateProfile', {
            email: this.profileForm.email
          })
          this.$message.success('保存成功')
        } catch (e) {
          // error handled by request interceptor
        }
      })
    },
    handleChangePassword() {
      this.$refs.pwdFormRef.validate(async (valid) => {
        if (!valid) return
        try {
          await this.$store.dispatch('user/changePassword', {
            old_password: this.pwdForm.old_password,
            new_password: this.pwdForm.new_password
          })
          this.$message.success('密码修改成功')
          this.pwdForm = { old_password: '', new_password: '', confirm_password: '' }
          this.$refs.pwdFormRef.resetFields()
        } catch (e) {
          // error handled by request interceptor
        }
      })
    }
  }
}
</script>
