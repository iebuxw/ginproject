import Vue from 'vue'
import Vuex from 'vuex'
import user from './modules/user'
import permission from './modules/permission'
import tagsView from './modules/tagsView'
import settings from './modules/settings'

Vue.use(Vuex)

export default new Vuex.Store({
  modules: { user, permission, tagsView, settings }
})
