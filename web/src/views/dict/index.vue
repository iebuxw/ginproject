<template>
  <div class="dict-container">
    <el-card shadow="never">
      <div slot="header" class="clearfix">
        <span>字典管理</span>
        <el-button type="primary" size="mini" style="float:right" @click="openTypeDialog()">新增</el-button>
      </div>
      <el-input v-model="typeKeyword" placeholder="输入编码或名称搜索" size="small" clearable @clear="fetchTypes" @keyup.enter.native="fetchTypes" style="margin-bottom:10px;width:240px">
        <el-button slot="append" icon="el-icon-search" @click="fetchTypes"></el-button>
      </el-input>
      <el-table :data="typeList" border v-loading="typeLoading">
        <el-table-column prop="id" label="ID" width="55" align="center"></el-table-column>
        <el-table-column prop="code" label="类型编码" show-overflow-tooltip></el-table-column>
        <el-table-column prop="name" label="类型名称" show-overflow-tooltip></el-table-column>
        <el-table-column label="状态" width="65" align="center">
          <template slot-scope="s">
            <el-tag :type="s.row.status === 1 ? 'success' : 'info'" size="mini">{{ s.row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="备注" show-overflow-tooltip></el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="160" align="center"></el-table-column>
        <el-table-column label="操作" width="250" align="center">
          <template slot-scope="s">
            <div style="white-space:nowrap">
              <el-button size="mini" @click="openTypeDialog(s.row)">编辑</el-button>
              <el-button size="mini" type="primary" plain @click="openDrawer(s.row)">字典项</el-button>
              <el-button size="mini" type="danger" @click="handleDeleteType(s.row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination small @current-change="typePageChange" :current-page="typePage" :page-size="typePageSize" :total="typeTotal" layout="total, prev, pager, next" style="margin-top:10px;text-align:right"></el-pagination>
    </el-card>

    <!-- 字典类型弹窗 -->
    <el-dialog :title="typeDialogTitle" :visible.sync="typeDialogVisible" width="480px" @close="resetTypeForm">
      <el-form :model="typeForm" label-width="80px" ref="typeForm">
        <el-form-item label="类型编码"><el-input v-model="typeForm.code" placeholder="请输入类型编码" :disabled="typeIsEdit"></el-input></el-form-item>
        <el-form-item label="类型名称"><el-input v-model="typeForm.name" placeholder="请输入类型名称"></el-input></el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="typeForm.status" :active-value="1" :inactive-value="0"></el-switch>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="typeForm.description" type="textarea" :rows="2" placeholder="请输入备注"></el-input></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="typeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmitType">确定</el-button>
      </span>
    </el-dialog>

    <!-- 字典项抽屉 -->
    <el-drawer :title="'字典项 — ' + (currentType ? currentType.name : '')" :visible.sync="drawerVisible" size="60%" :before-close="handleDrawerClose">
      <div class="drawer-content">
        <el-button type="primary" size="small" style="margin-bottom:15px" @click="openDataDialog()">新增字典项</el-button>
        <el-table :data="dataList" border v-loading="dataLoading">
          <el-table-column prop="label" label="标签" show-overflow-tooltip></el-table-column>
          <el-table-column prop="value" label="值" show-overflow-tooltip></el-table-column>
          <el-table-column prop="sort" label="排序" width="80" align="center"></el-table-column>
          <el-table-column label="状态" width="65" align="center">
            <template slot-scope="s">
              <el-tag :type="s.row.status === 1 ? 'success' : 'info'" size="mini">{{ s.row.status === 1 ? '启用' : '禁用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="remark" label="备注" show-overflow-tooltip></el-table-column>
          <el-table-column label="操作" width="150" align="center">
            <template slot-scope="s">
              <div style="white-space:nowrap">
                <el-button size="mini" @click="openDataDialog(s.row)">编辑</el-button>
                <el-button size="mini" type="danger" @click="handleDeleteData(s.row.id)">删除</el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination small @current-change="dataPageChange" :current-page="dataPage" :page-size="dataPageSize" :total="dataTotal" layout="total, prev, pager, next" style="margin-top:10px;text-align:right"></el-pagination>
      </div>
    </el-drawer>

    <!-- 字典数据弹窗 -->
    <el-dialog :title="dataDialogTitle" :visible.sync="dataDialogVisible" width="480px" @close="resetDataForm">
      <el-form :model="dataForm" label-width="80px" ref="dataForm">
        <el-form-item label="标签"><el-input v-model="dataForm.label" placeholder="请输入标签"></el-input></el-form-item>
        <el-form-item label="值"><el-input v-model="dataForm.value" placeholder="请输入值"></el-input></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="dataForm.sort" :min="0" :max="9999"></el-input-number></el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="dataForm.status" :active-value="1" :inactive-value="0"></el-switch>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="dataForm.remark" placeholder="请输入备注"></el-input></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dataDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmitData">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { getDictTypes, getDictType, addDictType, updateDictType, deleteDictType, getDictData, getDictDataById, addDictData, updateDictData, deleteDictData } from '@/api/dict'

export default {
  name: 'DictType',
  data() {
    return {
      // 字典类型
      typeList: [],
      typePage: 1,
      typePageSize: 10,
      typeTotal: 0,
      typeKeyword: '',
      typeLoading: false,
      currentType: null,
      // 类型弹窗
      typeDialogVisible: false,
      typeIsEdit: false,
      typeForm: { name: '', code: '', description: '', status: 1 },
      // 抽屉
      drawerVisible: false,
      // 字典数据
      dataList: [],
      dataPage: 1,
      dataPageSize: 10,
      dataTotal: 0,
      dataLoading: false,
      // 数据弹窗
      dataDialogVisible: false,
      dataIsEdit: false,
      dataForm: { label: '', value: '', sort: 0, remark: '', status: 1, dict_type_id: 0 }
    }
  },
  computed: {
    typeDialogTitle() { return this.typeIsEdit ? '编辑字典类型' : '新增字典类型' },
    dataDialogTitle() { return this.dataIsEdit ? '编辑字典项' : '新增字典项' }
  },
  created() {
    this.fetchTypes()
  },
  methods: {
    // ========== 字典类型 ==========
    async fetchTypes() {
      this.typeLoading = true
      try {
        const res = await getDictTypes({ page: this.typePage, page_size: this.typePageSize, keyword: this.typeKeyword })
        this.typeList = res.data.list
        this.typeTotal = res.data.total
      } finally {
        this.typeLoading = false
      }
    },
    typePageChange(p) {
      this.typePage = p
      this.fetchTypes()
    },
    openTypeDialog(row) {
      if (row) {
        this.typeIsEdit = true
        this.typeForm = { id: row.id, name: row.name, code: row.code, description: row.description, status: row.status }
      } else {
        this.typeIsEdit = false
        this.typeForm = { name: '', code: '', description: '', status: 1 }
      }
      this.typeDialogVisible = true
    },
    resetTypeForm() {
      this.typeForm = { name: '', code: '', description: '', status: 1 }
    },
    async handleSubmitType() {
      if (!this.typeForm.name || !this.typeForm.code) {
        this.$message.warning('类型编码和类型名称不能为空')
        return
      }
      if (this.typeIsEdit) {
        await updateDictType(this.typeForm.id, this.typeForm)
      } else {
        await addDictType(this.typeForm)
      }
      this.typeDialogVisible = false
      this.$message.success(this.typeIsEdit ? '编辑成功' : '新增成功')
      this.fetchTypes()
    },
    async handleDeleteType(row) {
      const res = await getDictData({ dict_type_id: row.id, page_size: 1 })
      if (res.data.total > 0) {
        this.$message.warning('该字典类型下还有数据，请先删除数据项')
        return
      }
      await this.$confirm('确认删除该字典类型?', '提示', { type: 'warning' })
      await deleteDictType(row.id)
      this.$message.success('删除成功')
      this.fetchTypes()
    },
    // ========== 字典项抽屉 ==========
    openDrawer(row) {
      this.currentType = row
      this.dataPage = 1
      this.drawerVisible = true
      this.fetchDataList()
    },
    handleDrawerClose(done) {
      done()
      this.currentType = null
      this.dataList = []
    },
    // ========== 字典数据 ==========
    async fetchDataList() {
      if (!this.currentType) return
      this.dataLoading = true
      try {
        const res = await getDictData({ page: this.dataPage, page_size: this.dataPageSize, dict_type_id: this.currentType.id })
        this.dataList = res.data.list
        this.dataTotal = res.data.total
      } finally {
        this.dataLoading = false
      }
    },
    dataPageChange(p) {
      this.dataPage = p
      this.fetchDataList()
    },
    openDataDialog(row) {
      if (row) {
        this.dataIsEdit = true
        this.dataForm = { id: row.id, label: row.label, value: row.value, sort: row.sort, remark: row.remark, status: row.status, dict_type_id: row.dict_type_id }
      } else {
        this.dataIsEdit = false
        this.dataForm = { label: '', value: '', sort: 0, remark: '', status: 1, dict_type_id: this.currentType.id }
      }
      this.dataDialogVisible = true
    },
    resetDataForm() {
      this.dataForm = { label: '', value: '', sort: 0, remark: '', status: 1, dict_type_id: this.currentType ? this.currentType.id : 0 }
    },
    async handleSubmitData() {
      if (!this.dataForm.label || !this.dataForm.value) {
        this.$message.warning('标签和值不能为空')
        return
      }
      if (this.dataIsEdit) {
        await updateDictData(this.dataForm.id, this.dataForm)
      } else {
        await addDictData(this.dataForm)
      }
      this.dataDialogVisible = false
      this.$message.success(this.dataIsEdit ? '编辑成功' : '新增成功')
      this.fetchDataList()
    },
    async handleDeleteData(id) {
      await this.$confirm('确认删除该字典项?', '提示', { type: 'warning' })
      await deleteDictData(id)
      this.$message.success('删除成功')
      this.fetchDataList()
    }
  }
}
</script>

<style scoped>
.dict-container {
  padding: 20px;
}
.drawer-content {
  padding: 0 20px 20px;
}
</style>
