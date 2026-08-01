import { login, logout, getUserInfo } from '@/api/auth'

const state = {
  token: localStorage.getItem('token') || '',
  userInfo: {}
}

const mutations = {
  SET_TOKEN(state, token) { state.token = token; localStorage.setItem('token', token) },
  SET_USER_INFO(state, info) { state.userInfo = info },
  CLEAR(state) { state.token = ''; state.userInfo = {}; localStorage.removeItem('token') }
}

const actions = {
  async login({ commit }, data) {
    const res = await login(data)
    commit('SET_TOKEN', res.data.token)
    return res
  },
  async getUserInfo({ commit }) {
    const res = await getUserInfo()
    commit('SET_USER_INFO', res.data)
    return res.data
  },
  async logout({ commit }) {
    await logout()
    commit('CLEAR')
  }
}

export default { namespaced: true, state, mutations, actions }
