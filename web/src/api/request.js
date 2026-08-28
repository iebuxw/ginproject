import axios from 'axios'
import store from '@/store'
import router from '@/router'
import { Message } from 'element-ui'

const request = axios.create({ baseURL: '/api', timeout: 15000 })

request.interceptors.request.use(config => {
  const token = store.state.user.token
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

let isLoggingOut = false

request.interceptors.response.use(
  response => {
    const res = response.data
    if (res.code !== 200) {
      Message.error(res.message || '请求失败')
      return Promise.reject(new Error(res.message))
    }
    return res
  },
  error => {
    if (error.response && error.response.status === 401 && !isLoggingOut) {
      isLoggingOut = true
      store.commit('user/CLEAR')
      router.push('/login')
      // 延迟重置标志，避免短时间内重复触发
      setTimeout(() => { isLoggingOut = false }, 1000)
    }
    return Promise.reject(error)
  }
)

export default request
