const state = {
  visitedViews: [],
  cachedViews: []
}

const mutations = {
  ADD_VISITED_VIEW(state, view) {
    if (state.visitedViews.some(v => v.path === view.path)) return
    state.visitedViews.push(
      Object.assign({}, view, {
        title: view.meta?.title || 'no-name'
      })
    )
  },
  ADD_CACHED_VIEW(state, view) {
    if (state.cachedViews.includes(view.name)) return
    if (!view.name) return
    state.cachedViews.push(view.name)
  },
  DEL_VISITED_VIEW(state, view) {
    const index = state.visitedViews.findIndex(v => v.path === view.path)
    if (index > -1) {
      state.visitedViews.splice(index, 1)
    }
  },
  DEL_CACHED_VIEW(state, view) {
    const index = state.cachedViews.indexOf(view.name)
    if (index > -1) {
      state.cachedViews.splice(index, 1)
    }
  },
  DEL_ALL_VISITED_VIEWS(state) {
    state.visitedViews = state.visitedViews.filter(v => v.meta?.affix)
  },
  DEL_ALL_CACHED_VIEWS(state) {
    state.cachedViews = []
  }
}

const actions = {
  addView({ commit }, view) {
    commit('ADD_VISITED_VIEW', view)
    commit('ADD_CACHED_VIEW', view)
  },
  addVisitedView({ commit }, view) {
    commit('ADD_VISITED_VIEW', view)
  },
  addCachedView({ commit }, view) {
    commit('ADD_CACHED_VIEW', view)
  },
  delView({ commit, state }, view) {
    return new Promise(resolve => {
      commit('DEL_VISITED_VIEW', view)
      commit('DEL_CACHED_VIEW', view)
      resolve({
        visitedViews: [...state.visitedViews],
        cachedViews: [...state.cachedViews]
      })
    })
  },
  delAllViews({ commit, state }) {
    return new Promise(resolve => {
      commit('DEL_ALL_VISITED_VIEWS')
      commit('DEL_ALL_CACHED_VIEWS')
      resolve({
        visitedViews: [...state.visitedViews],
        cachedViews: [...state.cachedViews]
      })
    })
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
