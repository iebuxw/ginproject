import Vue from 'vue'
import Router from 'vue-router'
import Layout from '@/layout/index.vue'
import store from '@/store'

Vue.use(Router)

export const constantRoutes = [
  {
    path: '/login',
    component: () => import('@/views/login/index.vue'),
    hidden: true
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        name: 'Dashboard',
        meta: { title: '控制台' }
      }
    ]
  }
]

const createRouter = () => new Router({ mode: 'history', routes: constantRoutes })

const router = createRouter()

router.beforeEach(async (to, from, next) => {
  if (to.path === '/login') { next(); return }

  const token = localStorage.getItem('token')
  if (!token) { next('/login'); return }

  const hasRoutes = store.state.permission.routes.length > 0
  if (!hasRoutes) {
    try {
      const data = await store.dispatch('user/getUserInfo')
      store.commit('permission/SET_MENUS', data.menus)
      store.commit('permission/SET_PERMISSIONS', data.permissions || [])
      const routes = await store.dispatch('permission/generateRoutes', data.menus)
      router.addRoutes(routes)
      router.addRoutes([{ path: '*', redirect: '/dashboard' }])
      next({ ...to, replace: true })
    } catch (e) {
      store.commit('user/CLEAR')
      next('/login')
    }
  } else {
    next()
  }
})

export default router
