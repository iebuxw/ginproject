<template>
  <div>
    <el-card>
      <div slot="header"><span>登录日志</span></div>
      <el-form :inline="true">
        <el-form-item>
          <el-input v-model="filters.username" placeholder="用户名" clearable @keyup.enter.native="fetchData" style="width:180px"></el-input>
        </el-form-item>
        <el-form-item>
          <el-select v-model="filters.status" placeholder="状态" clearable @change="fetchData">
            <el-option label="成功" :value="1"></el-option>
            <el-option label="失败" :value="0"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item><el-button @click="fetchData">查询</el-button></el-form-item>
      </el-form>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column prop="username" label="用户名" width="120"></el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '成功' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="原因" show-overflow-tooltip></el-table-column>
        <el-table-column prop="ip" label="IP" width="140"></el-table-column>
        <el-table-column prop="created_at" label="登录时间" width="180"></el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>
  </div>
</template>
<script>
import { getLoginLogs } from '@/api/log'
export default {
  data() {
    return {
      list: [], page: 1, pageSize: 10, total: 0,
      filters: { username: '', status: '' }
    }
  },
  created() { this.fetchData() },
  methods: {
    async fetchData() {
      const res = await getLoginLogs({ page: this.page, page_size: this.pageSize, username: this.filters.username, status: this.filters.status })
      this.list = res.data.list; this.total = res.data.total
    },
    pageChange(p) { this.page = p; this.fetchData() }
  }
}
</script>
