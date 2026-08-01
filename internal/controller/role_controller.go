package controller

import (
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RoleController struct{ roleService *service.RoleService }

func NewRoleController(roleService *service.RoleService) *RoleController {
	return &RoleController{roleService}
}

func (ctl *RoleController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	roles, total, err := ctl.roleService.FindPage(page, pageSize, keyword)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": roles, "total": total})
}

func (ctl *RoleController) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	role, err := ctl.roleService.FindByID(uint(id))
	if err != nil {
		utils.Error(c, 404, "角色不存在")
		return
	}
	var menuIDs []uint
	for _, m := range role.Menus {
		menuIDs = append(menuIDs, m.ID)
	}
	utils.Success(c, gin.H{"role": role, "menu_ids": menuIDs})
}

func (ctl *RoleController) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		MenuIDs     []uint `json:"menu_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	r := &model.Role{Name: req.Name, Code: req.Code, Description: req.Description, Status: 1}
	for _, id := range req.MenuIDs {
		r.Menus = append(r.Menus, model.Menu{ID: id})
	}
	if err := ctl.roleService.Create(r); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

func (ctl *RoleController) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		MenuIDs     []uint `json:"menu_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	r := &model.Role{ID: uint(id), Name: req.Name, Code: req.Code, Description: req.Description, Status: 1}
	for _, mid := range req.MenuIDs {
		r.Menus = append(r.Menus, model.Menu{ID: mid})
	}
	if err := ctl.roleService.Update(r); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

func (ctl *RoleController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.roleService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}
