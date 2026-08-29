const state = {
  unreadCount: 0
}

const mutations = {
  SET_UNREAD(state, n) { state.unreadCount = n },
  INC_UNREAD(state) { state.unreadCount++ },
  DEC_UNREAD(state) { if (state.unreadCount > 0) state.unreadCount-- },
  CLEAR_UNREAD(state) { state.unreadCount = 0 }
}

export default { namespaced: true, state, mutations }
