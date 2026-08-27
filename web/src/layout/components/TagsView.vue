<template>
  <div class="tags-view-container" @contextmenu.prevent="handleGlobalContextMenu">
    <div class="tags-view-wrapper" ref="scrollContainer">
      <router-link
        v-for="tag in visitedViews"
        :key="tag.path"
        :to="{ path: tag.path, query: tag.query, fullPath: tag.fullPath }"
        :class="isActive(tag) ? 'active' : ''"
        class="tags-view-item"
        @contextmenu.prevent.stop="openMenu(tag, $event)"
      >
        <i v-if="tag.meta && tag.meta.icon" :class="tag.meta.icon" class="tag-icon"></i>
        {{ tag.title }}
        <span
          v-if="!isAffix(tag)"
          class="el-icon-close tag-close"
          @click.prevent.stop="closeTag(tag)"
        ></span>
      </router-link>
    </div>
    <ul
      v-show="visible"
      :style="{ left: left + 'px', top: top + 'px' }"
      class="contextmenu"
    >
      <li @click="refreshTag(selectedTag)">
        <i class="el-icon-refresh"></i> 刷新
      </li>
      <li v-if="!isAffix(selectedTag)" @click="closeTag(selectedTag)">
        <i class="el-icon-close"></i> 关闭
      </li>
      <li @click="closeOthersTags">
        <i class="el-icon-circle-close"></i> 关闭其他
      </li>
      <li @click="closeAllTags">
        <i class="el-icon-folder-remove"></i> 关闭全部
      </li>
    </ul>
  </div>
</template>

<script>
import { mapState } from 'vuex'

export default {
  data() {
    return {
      visible: false,
      top: 0,
      left: 0,
      selectedTag: {}
    }
  },
  computed: {
    ...mapState('tagsView', ['visitedViews']),
    routes() {
      return this.$store.state.permission.routes
    }
  },
  watch: {
    visible(value) {
      if (value) {
        document.body.addEventListener('click', this.closeMenu)
      } else {
        document.body.removeEventListener('click', this.closeMenu)
      }
    }
  },
  mounted() {
    this.initTags()
    this.addTags()
  },
  beforeDestroy() {
    document.body.removeEventListener('click', this.closeMenu)
  },
  methods: {
    isActive(route) {
      return route.path === this.$route.path
    },
    isAffix(tag) {
      return tag.meta && tag.meta.affix
    },
    filterAffixTags(routes, basePath = '/') {
      let tags = []
      routes.forEach(route => {
        if (route.meta && route.meta.affix) {
          const tagPath = basePath + route.path
          tags.push({
            fullPath: tagPath,
            path: tagPath,
            name: route.name,
            meta: { ...route.meta }
          })
        }
        if (route.children) {
          const tempTags = this.filterAffixTags(route.children, route.path)
          if (tempTags.length >= 1) {
            tags = [...tags, ...tempTags]
          }
        }
      })
      return tags
    },
    initTags() {
      const affixTags = this.filterAffixTags(this.routes)
      for (const tag of affixTags) {
        if (tag.name) {
          this.$store.dispatch('tagsView/addVisitedView', tag)
        }
      }
    },
    addTags() {
      const { name } = this.$route
      if (name) {
        this.$store.dispatch('tagsView/addView', this.$route)
      }
    },
    refreshTag(tag) {
      this.$store.dispatch('tagsView/delCachedView', tag).then(() => {
        const { fullPath } = tag
        this.$nextTick(() => {
          this.$router.replace({ path: '/redirect' + fullPath })
        })
      })
      this.closeMenu()
    },
    closeTag(tag) {
      this.$store.dispatch('tagsView/delView', tag).then(visitedViews => {
        if (this.isActive(tag)) {
          this.toLastView(visitedViews)
        }
      })
      this.closeMenu()
    },
    closeOthersTags() {
      this.$router.push(this.selectedTag)
      this.$store.dispatch('tagsView/delOthersViews', this.selectedTag)
      this.closeMenu()
    },
    closeAllTags() {
      this.$store.dispatch('tagsView/delAllViews').then(visitedViews => {
        if (this.affixTags.some(tag => tag.path === this.$route.path)) {
          return
        }
        this.toLastView(visitedViews)
      })
      this.closeMenu()
    },
    toLastView(visitedViews) {
      const latestView = visitedViews.slice(-1)[0]
      if (latestView) {
        this.$router.push(latestView.fullPath)
      } else {
        this.$router.push('/')
      }
    },
    openMenu(tag, e) {
      const menuMinWidth = 105
      const offsetLeft = this.$el.getBoundingClientRect().left
      const offsetWidth = this.$el.offsetWidth
      const maxLeft = offsetWidth - menuMinWidth
      const left = e.clientX - offsetLeft + 15
      if (left > maxLeft) {
        this.left = maxLeft
      } else {
        this.left = left
      }
      this.top = e.clientY
      this.visible = true
      this.selectedTag = tag
    },
    handleGlobalContextMenu() {
      this.closeMenu()
    },
    closeMenu() {
      this.visible = false
    }
  }
}
</script>

<style scoped>
.tags-view-container {
  height: 34px;
  width: 100%;
  background: #fff;
  border-bottom: 1px solid #e6e6e6;
  position: relative;
}

.tags-view-wrapper {
  height: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  white-space: nowrap;
  padding: 0 10px;
}

.tags-view-wrapper::-webkit-scrollbar {
  height: 4px;
}

.tags-view-wrapper::-webkit-scrollbar-thumb {
  background: #d8dce5;
  border-radius: 2px;
}

.tags-view-wrapper::-webkit-scrollbar-track {
  background: transparent;
}

.tags-view-item {
  display: inline-flex;
  align-items: center;
  position: relative;
  cursor: pointer;
  height: 26px;
  line-height: 26px;
  border: 1px solid #d8dce5;
  color: #495060;
  background: #fff;
  padding: 0 8px;
  font-size: 12px;
  margin-left: 5px;
  margin-top: 4px;
  text-decoration: none;
  border-radius: 2px;
}

.tags-view-item:first-of-type {
  margin-left: 0;
}

.tags-view-item:hover {
  color: #409eff;
}

.tags-view-item.active {
  background-color: #409eff;
  color: #fff;
  border-color: #409eff;
}

.tags-view-item.active .tag-close {
  color: #fff;
}

.tags-view-item.active .tag-close:hover {
  background-color: rgba(255, 255, 255, 0.3);
  border-radius: 50%;
}

.tag-icon {
  margin-right: 4px;
  font-size: 12px;
}

.tag-close {
  width: 16px;
  height: 16px;
  line-height: 16px;
  text-align: center;
  border-radius: 50%;
  margin-left: 4px;
  transition: all 0.3s;
}

.tag-close:hover {
  background-color: #b4bccc;
  color: #fff;
}

.contextmenu {
  margin: 0;
  background: #fff;
  z-index: 3000;
  position: absolute;
  list-style-type: none;
  padding: 5px 0;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 400;
  color: #333;
  box-shadow: 2px 2px 3px 0 rgba(0, 0, 0, 0.3);
}

.contextmenu li {
  margin: 0;
  padding: 7px 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
}

.contextmenu li:hover {
  background: #eee;
}

.contextmenu li i {
  margin-right: 6px;
}
</style>
