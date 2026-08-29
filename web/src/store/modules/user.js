import { login, logout, getUserInfo, changePassword, updateProfile } from '@/api/auth'
import { connectWS } from '@/utils/ws'

const state = {
  token: localStorage.getItem('token') || '',
  userInfo: {}
}

const mutations = {
  SET_TOKEN(state, token) { state.token = token; localStorage.setItem('token', token) },
  SET_USER_INFO(state, info) { state.userInfo = info },
  SET_AVATAR(state, url) { state.userInfo.avatar = url },
  CLEAR(state) { state.token = ''; state.userInfo = {}; localStorage.removeItem('token') }
}

const actions = {
  async login({ commit }, data) {
    const res = await login(data)
    commit('SET_TOKEN', res.data.token)
    commit('SET_USER_INFO', res.data.user)
    connectWS(res.data.token)
    return res
  },
  async getUserInfo({ commit }) {
    const res = await getUserInfo()
    commit('SET_USER_INFO', res.data)
    return res.data
  },
  async changePassword(_, data) {
    await changePassword(data)
  },
  async updateProfile({ commit }, data) {
    await updateProfile(data)
    commit('SET_USER_INFO', { ...state.userInfo, ...data })
  },
  async logout({ commit }) {
    try {
      await logout()
    } catch (e) {
      // token 已过期或网络异常，忽略错误，仍清除本地状态
    }
    commit('CLEAR')
  }
}

export default { namespaced: true, state, mutations, actions }
