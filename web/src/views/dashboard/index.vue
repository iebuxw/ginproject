<template>
  <div>
    <div class="refresh-hint">每 10 秒自动刷新</div>
    <div class="page-title">服务器状态</div>

    <div class="stat-cards">
      <div class="stat-card">
        <div class="progress-wrap">
          <el-progress type="circle" :percentage="info.cpu.usage_percent" :width="90" :stroke-width="8"
            :color="'#409EFF'" :format="p => p.toFixed(1) + '%'"></el-progress>
        </div>
        <div class="info">
          <div class="label">CPU 使用率</div>
          <div class="value">{{ info.cpu.usage_percent.toFixed(1) }}%</div>
          <div class="sub">{{ info.cpu.cores }} 核心</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="progress-wrap">
          <el-progress type="circle" :percentage="info.memory.usage_percent" :width="90" :stroke-width="8"
            :color="'#E6A23C'" :format="p => p.toFixed(1) + '%'"></el-progress>
        </div>
        <div class="info">
          <div class="label">内存使用率</div>
          <div class="value">{{ info.memory.usage_percent.toFixed(1) }}%</div>
          <div class="sub">{{ formatBytes(info.memory.used) }} / {{ formatBytes(info.memory.total) }}</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="progress-wrap">
          <el-progress type="circle" :percentage="info.disk.usage_percent" :width="90" :stroke-width="8"
            :color="'#67C23A'" :format="p => p.toFixed(1) + '%'"></el-progress>
        </div>
        <div class="info">
          <div class="label">磁盘使用率</div>
          <div class="value">{{ info.disk.usage_percent.toFixed(1) }}%</div>
          <div class="sub">{{ formatBytes(info.disk.used) }} / {{ formatBytes(info.disk.total) }}</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="progress-wrap">
          <el-progress type="circle" :percentage="goroutinePercent" :width="90" :stroke-width="8"
            :color="'#909399'" :format="() => info.runtime.goroutines"></el-progress>
        </div>
        <div class="info">
          <div class="label">Goroutines</div>
          <div class="value">{{ info.runtime.goroutines }}</div>
          <div class="sub">{{ info.runtime.go_version }}</div>
        </div>
      </div>
    </div>

    <div class="detail-card">
      <div class="el-card__header">运行时信息</div>
      <div class="detail-items">
        <div class="detail-item">
          <div class="label">Go 版本</div>
          <div class="val">{{ info.runtime.go_version }}</div>
        </div>
        <div class="detail-item">
          <div class="label">操作系统</div>
          <div class="val">{{ info.runtime.os }} / {{ info.runtime.arch }}</div>
        </div>
        <div class="detail-item">
          <div class="label">进程内存分配</div>
          <div class="val">{{ formatBytes(info.runtime.mem_alloc) }}</div>
        </div>
        <div class="detail-item">
          <div class="label">进程运行时间</div>
          <div class="val">{{ formatUptime(info.runtime.uptime) }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { getServerInfo } from '@/api/dashboard'

export default {
  name: 'Dashboard',
  data() {
    return {
      timer: null,
      info: {
        cpu: { usage_percent: 0, cores: 0 },
        memory: { total: 0, used: 0, usage_percent: 0 },
        disk: { total: 0, used: 0, usage_percent: 0 },
        runtime: { go_version: '', os: '', arch: '', goroutines: 0, mem_alloc: 0, uptime: 0 }
      }
    }
  },
  computed: {
    goroutinePercent() {
      const max = 500
      return Math.min(Math.round((this.info.runtime.goroutines / max) * 100), 100)
    }
  },
  created() {
    this.fetchInfo()
    this.timer = setInterval(this.fetchInfo, 10000)
  },
  beforeDestroy() {
    if (this.timer) clearInterval(this.timer)
  },
  methods: {
    async fetchInfo() {
      try {
        const res = await getServerInfo()
        if (res.code === 200) {
          this.info = res.data
        }
      } catch (e) {
        // 静默失败，保持上次数据
      }
    },
    formatBytes(bytes) {
      if (bytes === 0) return '0 B'
      const units = ['B', 'KB', 'MB', 'GB', 'TB']
      const i = Math.floor(Math.log(bytes) / Math.log(1024))
      return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
    },
    formatUptime(seconds) {
      if (!seconds) return '-'
      const d = Math.floor(seconds / 86400)
      const h = Math.floor((seconds % 86400) / 3600)
      const m = Math.floor((seconds % 3600) / 60)
      if (d > 0) return d + ' 天 ' + h + ' 小时 ' + m + ' 分'
      if (h > 0) return h + ' 小时 ' + m + ' 分'
      return m + ' 分'
    }
  }
}
</script>

<style scoped>
.refresh-hint { text-align: right; font-size: 12px; color: #C0C4CC; margin-bottom: 8px; }
.page-title { font-size: 18px; font-weight: 600; color: #303133; margin-bottom: 20px; }
.stat-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 20px; }
.stat-card { background: #fff; border-radius: 4px; padding: 20px; display: flex; align-items: center; box-shadow: 0 1px 4px rgba(0,0,0,.08); }
.stat-card .progress-wrap { margin-right: 20px; flex-shrink: 0; }
.stat-card .info { flex: 1; }
.stat-card .label { font-size: 14px; color: #909399; margin-bottom: 8px; }
.stat-card .value { font-size: 24px; font-weight: 600; color: #303133; }
.stat-card .sub { font-size: 12px; color: #909399; margin-top: 4px; }
.detail-card { background: #fff; border-radius: 4px; box-shadow: 0 1px 4px rgba(0,0,0,.08); }
.detail-card .el-card__header { padding: 12px 20px; font-size: 15px; font-weight: 500; border-bottom: 1px solid #ebeef5; }
.detail-items { display: grid; grid-template-columns: repeat(4, 1fr); gap: 24px; padding: 20px; }
.detail-item .label { font-size: 13px; color: #909399; margin-bottom: 6px; }
.detail-item .val { font-size: 16px; font-weight: 500; color: #303133; }
</style>
