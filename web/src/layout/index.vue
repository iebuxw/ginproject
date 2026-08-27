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
          <span style="cursor:pointer;display:inline-flex;align-items:center">
            <el-avatar :size="30" :src="userInfo.avatar" style="margin-right:8px">
              {{ (userInfo.username || '用户').charAt(0) }}
            </el-avatar>
            {{ userInfo.username || '用户' }} <i class="el-icon-arrow-down"></i>
          </span>
          <el-dropdown-menu slot="dropdown">
            <el-dropdown-item command="profile">个人中心</el-dropdown-item>
            <el-dropdown-item command="logout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </el-dropdown>
      </el-header>
      <tags-view />
      <el-main>
        <keep-alive :include="cachedViews">
          <router-view :key="key" />
        </keep-alive>
      </el-main>
    </el-container>

  </el-container>
</template>

<script>
import { mapState } from 'vuex'
import { connectWS, disconnectWS } from '@/utils/ws'
import TagsView from './components/TagsView.vue'
export default {
  components: { TagsView },
  data() {
    return {
      menus: []
    }
  },
  computed: {
    ...mapState('user', ['userInfo']),
    ...mapState('tagsView', ['cachedViews']),
    key() {
      return this.$route.path
    }
  },
  created() {
    this.menus = this.$store.state.permission.menus
    const token = this.$store.state.user.token
    if (token) connectWS(token)
  },
  beforeDestroy() {
    disconnectWS()
  },
  methods: {
    hasVisibleChildren(item) {
      return item.children && item.children.some(c => c.type === 2)
    },
    async handleCommand(cmd) {
      if (cmd === 'profile') {
        this.$router.push('/profile')
      } else if (cmd === 'logout') {
        disconnectWS()
        await this.$store.dispatch('user/logout')
        this.$router.push('/login')
      }
    }
  }
}
</script>
