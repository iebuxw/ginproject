package controller

import (
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserController struct{ userService *service.UserService }

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService}
}

// List 获取用户分页列表
// @Summary 获取用户分页列表
// @Description 分页查询用户列表，支持关键词搜索
// @Tags 用户管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词（用户名/邮箱）"
// @Success 200 {object} utils.Response{data=object{list=[]model.User,total=int}} "成功"
// @Router /users [get]
func (ctl *UserController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	users, total, err := ctl.userService.FindPage(page, pageSize, keyword)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": users, "total": total})
}

// Get 获取用户详情
// @Summary 获取用户详情
// @Description 根据 ID 查询用户详情
// @Tags 用户管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "用户 ID"
// @Success 200 {object} utils.Response{data=model.User} "成功"
// @Failure 200 {object} utils.Response "用户不存在"
// @Router /users/{id} [get]
func (ctl *UserController) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user, err := ctl.userService.FindByID(uint(id))
	if err != nil {
		utils.Error(c, 404, "用户不存在")
		return
	}
	utils.Success(c, user)
}

// Create 新建用户
// @Summary 新建用户
// @Description 创建新用户并分配角色
// @Tags 用户管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object true "用户信息"
// @Param body.body.username body string true "用户名"
// @Param body.body.password body string true "密码"
// @Param body.body.email body string false "邮箱"
// @Param body.body.phone body string false "手机号"
// @Param body.body.status body int false "状态 1=启用 0=禁用" default(1)
// @Param body.body.role_ids body []uint false "角色 ID 列表"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /users [post]
func (ctl *UserController) Create(c *gin.Context) {
	var req struct {
		model.User
		RoleIDs []uint `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.userService.Create(&req.User, req.RoleIDs); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Update 编辑用户
// @Summary 编辑用户
// @Description 更新用户信息，密码为空时不覆盖
// @Tags 用户管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "用户 ID"
// @Param body body object true "用户信息"
// @Param body.body.username body string true "用户名"
// @Param body.body.password body string false "密码（为空不修改）"
// @Param body.body.email body string false "邮箱"
// @Param body.body.phone body string false "手机号"
// @Param body.body.status body int false "状态 1=启用 0=禁用"
// @Param body.body.role_ids body []uint false "角色 ID 列表"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /users/{id} [put]
func (ctl *UserController) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		model.User
		RoleIDs []uint `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	req.User.ID = uint(id)
	if err := ctl.userService.Update(&req.User, req.RoleIDs); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Delete 删除用户
// @Summary 删除用户
// @Description 根据 ID 删除用户
// @Tags 用户管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "用户 ID"
// @Success 200 {object} utils.Response "成功"
// @Router /users/{id} [delete]
func (ctl *UserController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.userService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}
