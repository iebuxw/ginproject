import Layout from '@/layout/index.vue'

// 菜单 path → 前端组件映射；新增菜单页必须在此添加
const componentMap = {
  '/system/user': () => import('@/views/user/index.vue'),
  '/system/role': () => import('@/views/role/index.vue'),
  '/system/menu': () => import('@/views/menu/index.vue'),
  '/system/log': () => import('@/views/log/index.vue'),
  '/system/login-log': () => import('@/views/loginlog/index.vue'),
  '/system/dict-type': () => import('@/views/dict/index.vue'),
  '/system/task': () => import('@/views/task/index.vue'),
  '/system/task-logs': () => import('@/views/task/logs.vue'),
  '/system/backup': () => import('@/views/backup/index.vue'),
  '/system/file': () => import('@/views/file/index.vue'),
  '/system/setting': () => import('@/views/setting/index.vue'),
  '/system/notification': () => import('@/views/notification/index.vue'),
  '/system/notification-send': () => import('@/views/notification/send.vue')
}

const state = {
  routes: [],
  menus: [],
  permissions: []
}

const mutations = {
  SET_ROUTES(state, routes) { state.routes = routes },
  SET_MENUS(state, menus) { state.menus = menus },
  SET_PERMISSIONS(state, perms) { state.permissions = perms }
}

const actions = {
  generateRoutes({ commit }, menus) {
    const routes = []
    menus.forEach(item => {
      if (item.type === 1 && item.children) {// type=1 是目录
        item.children.forEach(child => {
          if (child.type === 2) {           // type=2 是页面菜单
            const comp = componentMap[child.path] // ← 用菜单的 path 查映射表
            if (comp) {                 // 查到才生成路由，查不到就跳过
              routes.push({
                path: child.path,
                component: Layout,
                redirect: child.path,
                children: [{
                  path: '',
                  component: comp,
                  name: child.name,
                  meta: { title: child.name, icon: child.icon }
                }]
              })
            }
          }
        })
      }
    })
    commit('SET_ROUTES', routes)
    return routes
  }
}

export default { namespaced: true, state, mutations, actions }
