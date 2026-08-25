<template>
  <div class="dict-container">
    <!-- 左侧：字典类型 -->
    <div class="dict-left">
      <el-card shadow="never">
        <div slot="header" class="clearfix">
          <span>字典类型</span>
          <el-button type="primary" size="mini" style="float:right" @click="openTypeDialog()">新增</el-button>
        </div>
        <el-input v-model="typeKeyword" placeholder="搜索名称/编码" size="small" clearable @clear="fetchTypes" @keyup.enter.native="fetchTypes" style="margin-bottom:10px">
          <el-button slot="append" icon="el-icon-search" @click="fetchTypes"></el-button>
        </el-input>
        <el-table :data="typeList" border highlight-current-row size="small" @current-change="handleTypeChange" v-loading="typeLoading">
          <el-table-column prop="name" label="名称" show-overflow-tooltip></el-table-column>
          <el-table-column prop="code" label="编码" show-overflow-tooltip></el-table-column>
          <el-table-column label="状态" width="65" align="center">
            <template slot-scope="s">
              <el-tag :type="s.row.status === 1 ? 'success' : 'info'" size="mini">{{ s.row.status === 1 ? '启' : '禁' }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination small @current-change="typePageChange" :current-page="typePage" :page-size="typePageSize" :total="typeTotal" layout="total, prev, pager, next" style="margin-top:10px;text-align:right"></el-pagination>
      </el-card>
    </div>

    <!-- 右侧：字典数据 -->
    <div class="dict-right">
      <el-card shadow="never">
        <div slot="header" class="clearfix">
          <span>字典数据 {{ currentType ? '- ' + currentType.name : '' }}</span>
          <el-button type="primary" size="mini" style="float:right" :disabled="!currentType" @click="openDataDialog()">新增</el-button>
        </div>
        <el-table :data="dataList" border size="small" v-loading="dataLoading">
          <el-table-column prop="id" label="ID" width="55" align="center"></el-table-column>
          <el-table-column prop="label" label="显示文本" show-overflow-tooltip></el-table-column>
          <el-table-column prop="value" label="存储值" show-overflow-tooltip></el-table-column>
          <el-table-column prop="sort" label="排序" width="60" align="center"></el-table-column>
          <el-table-column label="状态" width="65" align="center">
            <template slot-scope="s">
              <el-tag :type="s.row.status === 1 ? 'success' : 'info'" size="mini">{{ s.row.status === 1 ? '启' : '禁' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="remark" label="备注" show-overflow-tooltip></el-table-column>
          <el-table-column label="操作" width="130" align="center">
            <template slot-scope="s">
              <el-button size="mini" @click="openDataDialog(s.row)">编辑</el-button>
              <el-button size="mini" type="danger" @click="handleDeleteData(s.row.id)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination small @current-change="dataPageChange" :current-page="dataPage" :page-size="dataPageSize" :total="dataTotal" layout="total, prev, pager, next" style="margin-top:10px;text-align:right"></el-pagination>
      </el-card>
    </div>

    <!-- 字典类型弹窗 -->
    <el-dialog :title="typeDialogTitle" :visible.sync="typeDialogVisible" width="480px" @close="resetTypeForm">
      <el-form :model="typeForm" label-width="80px" ref="typeForm">
        <el-form-item label="名称"><el-input v-model="typeForm.name" placeholder="请输入字典名称"></el-input></el-form-item>
        <el-form-item label="编码"><el-input v-model="typeForm.code" placeholder="请输入字典编码" :disabled="typeIsEdit"></el-input></el-form-item>
        <el-form-item label="描述"><el-input v-model="typeForm.description" type="textarea" :rows="2"></el-input></el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="typeForm.status" :active-value="1" :inactive-value="0"></el-switch>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="typeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmitType">确定</el-button>
      </span>
    </el-dialog>

    <!-- 字典数据弹窗 -->
    <el-dialog :title="dataDialogTitle" :visible.sync="dataDialogVisible" width="480px" @close="resetDataForm">
      <el-form :model="dataForm" label-width="80px" ref="dataForm">
        <el-form-item label="显示文本"><el-input v-model="dataForm.label" placeholder="请输入显示文本"></el-input></el-form-item>
        <el-form-item label="存储值"><el-input v-model="dataForm.value" placeholder="请输入存储值"></el-input></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="dataForm.sort" :min="0" :max="9999"></el-input-number></el-form-item>
        <el-form-item label="备注"><el-input v-model="dataForm.remark" placeholder="请输入备注"></el-input></el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="dataForm.status" :active-value="1" :inactive-value="0"></el-switch>
        </el-form-item>
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
    dataDialogTitle() { return this.dataIsEdit ? '编辑字典数据' : '新增字典数据' }
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
    async handleTypeChange(row) {
      this.currentType = row
      this.dataPage = 1
      await this.fetchDataList()
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
        this.$message.warning('名称和编码不能为空')
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
        this.$message.warning('显示文本和存储值不能为空')
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
      await this.$confirm('确认删除该字典数据?', '提示', { type: 'warning' })
      await deleteDictData(id)
      this.$message.success('删除成功')
      this.fetchDataList()
    }
  }
}
</script>

<style scoped>
.dict-container {
  display: flex;
  height: calc(100vh - 84px);
}
.dict-left {
  width: 380px;
  flex-shrink: 0;
  margin-right: 10px;
}
.dict-left .el-card {
  height: 100%;
}
.dict-left >>> .el-card__body {
  display: flex;
  flex-direction: column;
  height: calc(100% - 56px);
}
.dict-left >>> .el-table {
  flex: 1;
  overflow: auto;
}
.dict-right {
  flex: 1;
  min-width: 0;
}
.dict-right .el-card {
  height: 100%;
}
</style>
