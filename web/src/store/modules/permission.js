import Layout from '@/layout/index.vue'

const componentMap = {
  '/system/user': () => import('@/views/user/index.vue'),
  '/system/role': () => import('@/views/role/index.vue'),
  '/system/menu': () => import('@/views/menu/index.vue'),
  '/system/log': () => import('@/views/log/index.vue'),
  '/system/login-log': () => import('@/views/loginlog/index.vue'),
  '/system/dict-type': () => import('@/views/dict/index.vue')
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
      if (item.type === 1 && item.children) {
        item.children.forEach(child => {
          if (child.type === 2) {
            const comp = componentMap[child.path]
            if (comp) {
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
