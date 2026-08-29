package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type RoleController struct{ roleService *service.RoleService }

func NewRoleController(roleService *service.RoleService) *RoleController {
	return &RoleController{roleService}
}

// List 获取角色分页列表
// @Summary 获取角色分页列表
// @Description 分页查询角色列表，支持关键词搜索
// @Tags 角色管理
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词（角色名/编码）"
// @Success 200 {object} utils.Response{data=object{list=[]model.Role,total=int}} "成功"
// @Router /roles [get]
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

// Get 获取角色详情
// @Summary 获取角色详情
// @Description 根据 ID 查询角色详情，包含关联的菜单 ID 列表
// @Tags 角色管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "角色 ID"
// @Success 200 {object} utils.Response{data=object{role=model.Role,menu_ids=[]uint}} "成功"
// @Failure 200 {object} utils.Response "角色不存在"
// @Router /roles/{id} [get]
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

// Create 新建角色
// @Summary 新建角色
// @Description 创建新角色并分配菜单权限
// @Tags 角色管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object true "角色信息"
// @Param body.body.name body string true "角色名称"
// @Param body.body.code body string true "角色编码"
// @Param body.body.description body string false "描述"
// @Param body.body.menu_ids body []uint false "菜单 ID 列表"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /roles [post]
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

// Update 编辑角色
// @Summary 编辑角色
// @Description 更新角色信息及菜单权限
// @Tags 角色管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "角色 ID"
// @Param body body object true "角色信息"
// @Param body.body.name body string true "角色名称"
// @Param body.body.code body string true "角色编码"
// @Param body.body.description body string false "描述"
// @Param body.body.menu_ids body []uint false "菜单 ID 列表"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /roles/{id} [put]
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

// Delete 删除角色
// @Summary 删除角色
// @Description 根据 ID 删除角色
// @Tags 角色管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "角色 ID"
// @Success 200 {object} utils.Response "成功"
// @Router /roles/{id} [delete]
func (ctl *RoleController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.roleService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Export 导出角色列表为 Excel
// @Summary 导出角色列表
// @Description 同步导出角色数据为 Excel 文件
// @Tags 角色管理
// @Security BearerAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param keyword query string false "搜索关键词（角色名/标识）"
// @Success 200 {file} binary "Excel 文件"
// @Router /roles/export [get]
func (ctl *RoleController) Export(c *gin.Context) {
	keyword := c.Query("keyword")

	f := excelize.NewFile()
	defer f.Close()

	sheet := "角色列表"
	f.SetSheetName("Sheet1", sheet)

	sw, _ := f.NewStreamWriter(sheet)

	headers := []string{"ID", "角色名", "角色标识", "描述", "状态", "创建时间"}
	headerVals := make([]interface{}, len(headers))
	for i, h := range headers {
		headerVals[i] = h
	}
	sw.SetRow("A1", headerVals)

	offset := 0
	row := 2
	batchSize := 5000
	for {
		roles, err := ctl.roleService.FindBatch(keyword, offset, batchSize)
		if err != nil {
			utils.Error(c, 500, err.Error())
			return
		}
		if len(roles) == 0 {
			break
		}
		for _, r := range roles {
			cell, _ := excelize.CoordinatesToCellName(1, row)
			statusText := "启用"
			if r.Status != 1 {
				statusText = "禁用"
			}
			sw.SetRow(cell, []interface{}{
				r.ID, r.Name, r.Code, r.Description, statusText,
				time.Time(r.CreatedAt).Format("2006-01-02 15:04:05"),
			})
			row++
		}
		offset += batchSize
		if len(roles) < batchSize {
			break
		}
	}
	sw.Flush()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "F1", headerStyle)

	filename := fmt.Sprintf("角色列表_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(filename)))

	if _, err := f.WriteTo(c.Writer); err != nil {
		utils.ErrorWithStatus(c, http.StatusInternalServerError, 500, "导出失败")
	}
}
