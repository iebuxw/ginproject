package controller

import (
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MenuController struct{ menuService *service.MenuService }

func NewMenuController(menuService *service.MenuService) *MenuController {
	return &MenuController{menuService}
}

// List 获取菜单树
// @Summary 获取菜单树
// @Description 获取完整菜单树形结构
// @Tags 菜单管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} utils.Response{data=[]model.Menu} "成功"
// @Router /menus [get]
func (ctl *MenuController) List(c *gin.Context) {
	tree, err := ctl.menuService.GetTree()
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, tree)
}

// Get 获取菜单详情
// @Summary 获取菜单详情
// @Description 根据 ID 查询菜单详情
// @Tags 菜单管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "菜单 ID"
// @Success 200 {object} utils.Response{data=model.Menu} "成功"
// @Failure 200 {object} utils.Response "菜单不存在"
// @Router /menus/{id} [get]
func (ctl *MenuController) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	m, err := ctl.menuService.FindByID(uint(id))
	if err != nil {
		utils.Error(c, 404, "菜单不存在")
		return
	}
	utils.Success(c, m)
}

// Create 新建菜单
// @Summary 新建菜单
// @Description 创建新菜单（目录/菜单页/按钮权限）
// @Tags 菜单管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.Menu true "菜单信息"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /menus [post]
func (ctl *MenuController) Create(c *gin.Context) {
	var m model.Menu
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.menuService.Create(&m); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Update 编辑菜单
// @Summary 编辑菜单
// @Description 更新菜单信息
// @Tags 菜单管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "菜单 ID"
// @Param body body model.Menu true "菜单信息"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /menus/{id} [put]
func (ctl *MenuController) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var m model.Menu
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	m.ID = uint(id)
	if err := ctl.menuService.Update(&m); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Delete 删除菜单
// @Summary 删除菜单
// @Description 根据 ID 删除菜单，有子菜单时拒绝删除
// @Tags 菜单管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "菜单 ID"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "存在子菜单，无法删除"
// @Router /menus/{id} [delete]
func (ctl *MenuController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.menuService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Sort 批量更新菜单排序
// @Summary 批量更新菜单排序
// @Description 批量更新菜单的 sort 字段，用于拖拽排序
// @Tags 菜单管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body []model.MenuItemSort true "排序列表"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /menus/sort [put]
func (ctl *MenuController) Sort(c *gin.Context) {
	var items []model.MenuItemSort
	if err := c.ShouldBindJSON(&items); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.menuService.BatchUpdateSort(items); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}
