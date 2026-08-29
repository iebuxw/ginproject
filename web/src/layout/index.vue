<template>
  <el-container style="height:100vh">
    <el-aside width="220px" style="background:#304156;overflow-y:auto">
      <div style="color:#fff;text-align:center;line-height:60px;font-size:20px;font-weight:bold;display:flex;align-items:center;justify-content:center">
        <img v-if="siteLogo" :src="siteLogo" style="height:28px;margin-right:8px;border-radius:4px" />
        {{ siteName }}
      </div>
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
        <el-popover placement="bottom" width="320" trigger="click" @show="fetchRecent">
          <div style="max-height:300px;overflow-y:auto">
            <div v-for="item in recentList" :key="item.id" style="padding:8px 0;border-bottom:1px solid #eee;cursor:pointer" @click="readOne(item)">
              <div :style="item.read_at ? '' : 'font-weight:bold'">{{ item.title }}</div>
              <div style="color:#909399;font-size:12px">{{ item.created_at }}</div>
            </div>
            <div v-if="recentList.length === 0" style="text-align:center;color:#909399;padding:15px">暂无未读消息</div>
          </div>
          <div style="text-align:center;margin-top:8px">
            <el-button type="text" size="mini" @click="readAll">全部已读</el-button>
            <el-button type="text" size="mini" @click="goCenter">查看全部</el-button>
          </div>
          <el-badge slot="reference" :value="unreadCount" :max="99" :hidden="unreadCount === 0" class="item">
            <i class="el-icon-bell" style="font-size:20px;cursor:pointer"></i>
          </el-badge>
        </el-popover>
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
import { connectWS, disconnectWS, onWSMessage, offWSMessage } from '@/utils/ws'
import { getUnreadCount, getMyNotifications, markRead } from '@/api/notification'
import { getSettings } from '@/api/setting'
import TagsView from './components/TagsView.vue'
export default {
  components: { TagsView },
  data() {
    return {
      menus: [],
      recentList: []
    }
  },
  computed: {
    ...mapState('user', ['userInfo']),
    ...mapState('tagsView', ['cachedViews']),
    ...mapState('settings', { siteName: 'siteName', siteLogo: 'siteLogo' }),
    ...mapState('notification', ['unreadCount']),
    key() {
      return this.$route.path
    }
  },
  created() {
    this.menus = this.$store.state.permission.menus
    const token = this.$store.state.user.token
    if (token) connectWS(token)
    this.fetchUnread()
    this._onNotify = () => { this.fetchUnread() }
    onWSMessage('notification', this._onNotify)
    this.fetchSettings()
  },
  beforeDestroy() {
    offWSMessage('notification', this._onNotify)
    disconnectWS()
  },
  methods: {
    hasVisibleChildren(item) {
      return item.children && item.children.some(c => c.type === 2)
    },
    async fetchSettings() {
      try {
        const res = await getSettings()
        if (res.code === 200 && res.data) {
          const siteName = res.data.site_name || 'GinAdmin'
          const siteLogo = res.data.site_logo || ''
          this.$store.commit('settings/SET_SETTINGS', { siteName, siteLogo })
          document.title = siteName
        }
      } catch (e) {
        // 配置加载失败使用默认值
      }
    },
    async fetchUnread() {
      try {
        const res = await getUnreadCount()
        if (res.code === 200) this.$store.commit('notification/SET_UNREAD', res.data)
      } catch (e) { /* 静默 */ }
    },
    async fetchRecent() {
      try {
        const res = await getMyNotifications({ page: 1, page_size: 5, read_status: 1 })
        if (res.code === 200) this.recentList = res.data.list || []
      } catch (e) { /* 静默 */ }
    },
    async readOne(item) {
      try {
        await markRead({ ids: [item.id] })
        item.read_at = 'just-now'
        this.$store.commit('notification/DEC_UNREAD')
        this.fetchRecent()
      } catch (e) { /* 静默 */ }
    },
    async readAll() {
      try {
        await markRead({ all: true })
        this.$store.commit('notification/CLEAR_UNREAD')
        this.recentList = []
      } catch (e) { /* 静默 */ }
    },
    goCenter() {
      this.$router.push('/system/notification')
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
