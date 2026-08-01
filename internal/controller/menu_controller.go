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

func (ctl *MenuController) List(c *gin.Context) {
	tree, err := ctl.menuService.GetTree()
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, tree)
}

func (ctl *MenuController) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	m, err := ctl.menuService.FindByID(uint(id))
	if err != nil {
		utils.Error(c, 404, "菜单不存在")
		return
	}
	utils.Success(c, m)
}

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

func (ctl *MenuController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.menuService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}
