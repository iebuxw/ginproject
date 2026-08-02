<template>
  <el-container style="height:100vh">
    <el-aside width="220px" style="background:#304156;overflow-y:auto">
      <div style="color:#fff;text-align:center;line-height:60px;font-size:20px;font-weight:bold">GinAdmin</div>
      <el-menu
        :default-active="$route.path"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
        router
      >
        <template v-for="item in menus">
          <el-submenu v-if="item.children && hasVisibleChildren(item)" :key="item.id" :index="item.path">
            <template slot="title">
              <i :class="item.icon"></i>
              <span>{{ item.name }}</span>
            </template>
            <el-menu-item v-for="child in item.children" :key="child.id" :index="child.path" v-if="child.type === 2">
              <i :class="child.icon"></i>
              {{ child.name }}
            </el-menu-item>
          </el-submenu>
          <el-menu-item v-else-if="item.type === 2" :key="item.id" :index="item.path">
            <i :class="item.icon"></i>
            {{ item.name }}
          </el-menu-item>
        </template>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header style="background:#fff;line-height:60px;border-bottom:1px solid #e6e6e6;text-align:right;padding-right:20px">
        <el-dropdown @command="handleCommand">
          <span style="cursor:pointer">
            {{ userInfo.username || '用户' }} <i class="el-icon-arrow-down"></i>
          </span>
          <el-dropdown-menu slot="dropdown">
            <el-dropdown-item command="changePassword">修改密码</el-dropdown-item>
            <el-dropdown-item command="logout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </el-dropdown>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>

    <el-dialog title="修改密码" :visible.sync="pwdDialogVisible" width="400px">
      <el-form :model="pwdForm" label-width="80px">
        <el-form-item label="原密码"><el-input v-model="pwdForm.old_password" type="password"></el-input></el-form-item>
        <el-form-item label="新密码"><el-input v-model="pwdForm.new_password" type="password"></el-input></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="pwdDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleChangePassword">确定</el-button>
      </span>
    </el-dialog>
  </el-container>
</template>

<script>
import { mapState } from 'vuex'
export default {
  data() {
    return {
      menus: [],
      pwdDialogVisible: false,
      pwdForm: { old_password: '', new_password: '' }
    }
  },
  computed: { ...mapState('user', ['userInfo']) },
  created() {
    this.menus = this.$store.state.permission.menus
  },
  methods: {
    hasVisibleChildren(item) {
      return item.children && item.children.some(c => c.type === 2)
    },
    async handleCommand(cmd) {
      if (cmd === 'changePassword') {
        this.pwdDialogVisible = true
      } else if (cmd === 'logout') {
        await this.$store.dispatch('user/logout')
        this.$router.push('/login')
      }
    },
    async handleChangePassword() {
      if (!this.pwdForm.old_password || !this.pwdForm.new_password) {
        this.$message.warning('请输入密码')
        return
      }
      try {
        await this.$store.dispatch('user/changePassword', this.pwdForm)
        this.$message.success('密码修改成功')
        this.pwdDialogVisible = false
        this.pwdForm = { old_password: '', new_password: '' }
      } catch (e) {
        // error handled by request interceptor
      }
    }
  }
}
</script>
