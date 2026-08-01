<template>
  <div>
    <el-card>
      <div slot="header"><span>操作日志</span></div>
      <el-form :inline="true">
        <el-form-item>
          <el-select v-model="filters.method" placeholder="请求方式" clearable @change="fetchData">
            <el-option label="GET" value="GET"></el-option>
            <el-option label="POST" value="POST"></el-option>
            <el-option label="PUT" value="PUT"></el-option>
            <el-option label="DELETE" value="DELETE"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item><el-button @click="fetchData">查询</el-button></el-form-item>
      </el-form>
      <el-table :data="list" border>
        <el-table-column prop="id" label="ID" width="60"></el-table-column>
        <el-table-column prop="operator_id" label="操作人ID" width="80"></el-table-column>
        <el-table-column prop="method" label="方式" width="70"></el-table-column>
        <el-table-column prop="path" label="请求路径"></el-table-column>
        <el-table-column prop="duration" label="耗时(ms)" width="80"></el-table-column>
        <el-table-column prop="ip" label="IP" width="140"></el-table-column>
        <el-table-column prop="created_at" label="操作时间" width="180"></el-table-column>
      </el-table>
      <el-pagination style="margin-top:15px" @current-change="pageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total,prev,pager,next"></el-pagination>
    </el-card>
  </div>
</template>
<script>
import { getLogs } from '@/api/log'
export default {
  data() {
    return { list: [], page: 1, pageSize: 10, total: 0, filters: { method: '' } }
  },
  created() { this.fetchData() },
  methods: {
    async fetchData() {
      const res = await getLogs({ page: this.page, page_size: this.pageSize, method: this.filters.method })
      this.list = res.data.list; this.total = res.data.total
    },
    pageChange(p) { this.page = p; this.fetchData() }
  }
}
</script>
