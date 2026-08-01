<template>
  <div>
    <el-card>
      <div slot="header" class="clearfix">
        <span>菜单管理</span>
        <el-button type="primary" size="small" style="float:right" @click="openDialog()">新增菜单</el-button>
      </div>
      <el-table :data="menuList" border row-key="id" :tree-props="{ children: 'children' }" default-expand-all>
        <el-table-column prop="name" label="名称"></el-table-column>
        <el-table-column prop="icon" label="图标" width="80"></el-table-column>
        <el-table-column prop="path" label="路由路径"></el-table-column>
        <el-table-column label="类型" width="80">
          <template slot-scope="s">{{ ['', '目录', '菜单', '按钮'][s.row.type] }}</template>
        </el-table-column>
        <el-table-column prop="permission" label="权限标识"></el-table-column>
        <el-table-column prop="sort" label="排序" width="60"></el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="s">
            <el-tag :type="s.row.status === 1 ? 'success' : 'danger'">{{ s.row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template slot-scope="s">
            <el-button size="mini" @click="openDialog(s.row)">编辑</el-button>
            <el-button size="mini" type="danger" @click="handleDelete(s.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog :title="isEdit ? '编辑菜单' : '新增菜单'" :visible.sync="dialogVisible" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="上级菜单">
          <el-cascader
            v-model="form.parent_id"
            :options="cascaderOptions"
            :props="{ value: 'id', label: 'name', children: 'children', checkStrictly: true, emitPath: false }"
            clearable placeholder="不选=顶级菜单">
          </el-cascader>
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio :label="1">目录</el-radio>
            <el-radio :label="2">菜单</el-radio>
            <el-radio :label="3">按钮</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称"><el-input v-model="form.name"></el-input></el-form-item>
        <el-form-item label="图标" v-if="form.type < 3"><el-input v-model="form.icon"></el-input></el-form-item>
        <el-form-item label="路由路径" v-if="form.type < 3"><el-input v-model="form.path"></el-input></el-form-item>
        <el-form-item label="权限标识" v-if="form.type === 3"><el-input v-model="form.permission"></el-input></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0"></el-input-number></el-form-item>
        <el-form-item label="状态"><el-switch v-model="form.status" :active-value="1" :inactive-value="0"></el-switch></el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="handleSubmit">确定</el-button></span>
    </el-dialog>
  </div>
</template>
<script>
import { getMenus, addMenu, updateMenu, deleteMenu } from '@/api/menu'
export default {
  data() {
    return {
      menuList: [], dialogVisible: false, isEdit: false,
      form: { parent_id: 0, name: '', icon: '', path: '', type: 1, permission: '', sort: 0, status: 1 },
      cascaderOptions: []
    }
  },
  created() { this.fetchData() },
  methods: {
    async fetchData() {
      const res = await getMenus()
      this.menuList = res.data
      this.cascaderOptions = res.data
    },
    openDialog(row) {
      if (row) { this.isEdit = true; this.form = { ...row } }
      else { this.isEdit = false; this.form = { parent_id: 0, name: '', icon: '', path: '', type: 1, permission: '', sort: 0, status: 1 } }
      this.dialogVisible = true
    },
    async handleSubmit() {
      if (this.isEdit) { await updateMenu(this.form.id, this.form) }
      else { await addMenu(this.form) }
      this.dialogVisible = false; this.fetchData()
    },
    async handleDelete(id) {
      await this.$confirm('确认删除该菜单?', '提示', { type: 'warning' })
      try {
        await deleteMenu(id); this.fetchData(); this.$message.success('删除成功')
      } catch { this.$message.error('删除失败，可能存在子菜单') }
    }
  }
}
</script>
