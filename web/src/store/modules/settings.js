const state = {
  siteName: 'GinAdmin',
  siteLogo: ''
}

const mutations = {
  SET_SITE_NAME(state, name) { state.siteName = name },
  SET_SITE_LOGO(state, logo) { state.siteLogo = logo },
  SET_SETTINGS(state, { siteName, siteLogo }) {
    if (siteName !== undefined) state.siteName = siteName
    if (siteLogo !== undefined) state.siteLogo = siteLogo
  }
}

export default { namespaced: true, state, mutations }
