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

func (ctl *UserController) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user, err := ctl.userService.FindByID(uint(id))
	if err != nil {
		utils.Error(c, 404, "用户不存在")
		return
	}
	utils.Success(c, user)
}

func (ctl *UserController) Create(c *gin.Context) {
	var u model.User
	if err := c.ShouldBindJSON(&u); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.userService.Create(&u); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

func (ctl *UserController) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var u model.User
	if err := c.ShouldBindJSON(&u); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	u.ID = uint(id)
	if err := ctl.userService.Update(&u); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

func (ctl *UserController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.userService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}
