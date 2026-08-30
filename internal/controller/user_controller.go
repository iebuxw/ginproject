package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type UserController struct{ userService *service.UserService }

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService}
}

// List 获取管理员分页列表
// @Summary 获取管理员分页列表
// @Description 分页查询管理员列表，支持关键词搜索
// @Tags 管理员管理
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

// Get 获取管理员详情
// @Summary 获取管理员详情
// @Description 根据 ID 查询管理员详情
// @Tags 管理员管理
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

// Create 新建管理员
// @Summary 新建管理员
// @Description 创建新管理员并分配角色
// @Tags 管理员管理
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

// Update 编辑管理员
// @Summary 编辑管理员
// @Description 更新管理员信息，密码为空时不覆盖
// @Tags 管理员管理
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

// Delete 删除管理员
// @Summary 删除管理员
// @Description 根据 ID 删除管理员
// @Tags 管理员管理
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

// Export 导出管理员列表为 Excel
// @Summary 导出管理员列表
// @Description 同步导出管理员数据为 Excel 文件
// @Tags 管理员管理
// @Security BearerAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param keyword query string false "搜索关键词（用户名/邮箱）"
// @Success 200 {file} binary "Excel 文件"
// @Router /users/export [get]
func (ctl *UserController) Export(c *gin.Context) {
	keyword := c.Query("keyword")

	f := excelize.NewFile()
	defer f.Close()

	sheet := "管理员列表"
	f.SetSheetName("Sheet1", sheet)

	sw, _ := f.NewStreamWriter(sheet)

	headers := []string{"ID", "用户名", "邮箱", "手机号", "描述", "状态", "角色", "创建时间"}
	headerVals := make([]interface{}, len(headers))
	for i, h := range headers {
		headerVals[i] = h
	}
	sw.SetRow("A1", headerVals)

	offset := 0
	row := 2
	batchSize := 5000
	for {
		users, err := ctl.userService.FindBatch(keyword, offset, batchSize)
		if err != nil {
			utils.Error(c, 500, err.Error())
			return
		}
		if len(users) == 0 {
			break
		}
		for _, u := range users {
			cell, _ := excelize.CoordinatesToCellName(1, row)
			roleNames := ""
			for i, r := range u.Roles {
				if i > 0 {
					roleNames += ","
				}
				roleNames += r.Name
			}
			statusText := "启用"
			if u.Status != 1 {
				statusText = "禁用"
			}
			sw.SetRow(cell, []interface{}{
				u.ID, u.Username, u.Email, u.Phone,
				u.Description, statusText, roleNames,
				time.Time(u.CreatedAt).Format("2006-01-02 15:04:05"),
			})
			row++
		}
		offset += batchSize
		if len(users) < batchSize {
			break
		}
	}
	sw.Flush()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "H1", headerStyle)

	filename := fmt.Sprintf("管理员列表_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(filename)))

	if _, err := f.WriteTo(c.Writer); err != nil {
		utils.ErrorWithStatus(c, http.StatusInternalServerError, 500, "导出失败")
	}
}

// Import 导入管理员
// @Summary Excel 批量导入管理员
// @Description 上传 xlsx 文件批量创建管理员；用户名已存在的行跳过，校验失败的行返回原因
// @Tags 管理员管理
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "xlsx 文件"
// @Success 200 {object} utils.Response{data=service.ImportResult} "导入结果汇总"
// @Failure 200 {object} utils.Response "业务错误"
// @Router /users/import [post]
func (ctl *UserController) Import(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		utils.Error(c, 400, "请选择文件")
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		utils.Error(c, 400, "文件格式错误，仅支持 xlsx")
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		utils.Error(c, 400, "Excel 无工作表")
		return
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		utils.Error(c, 400, "读取 Excel 失败")
		return
	}
	if len(rows) < 2 {
		utils.Error(c, 400, "Excel 无数据行")
		return
	}

	colIdx := make(map[string]int)
	for i, h := range rows[0] {
		colIdx[strings.TrimSpace(h)] = i
	}
	for _, required := range []string{"用户名", "密码"} {
		if _, ok := colIdx[required]; !ok {
			utils.Error(c, 400, "缺少必需列: "+required)
			return
		}
	}

	cell := func(row []string, header string) string {
		i, ok := colIdx[header]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	importRows := make([]service.ImportRow, 0, len(rows)-1)
	for i, row := range rows[1:] {
		empty := true
		for _, v := range row {
			if strings.TrimSpace(v) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		status := 1
		if cell(row, "状态") == "禁用" {
			status = 0
		}
		importRows = append(importRows, service.ImportRow{
			Username:    cell(row, "用户名"),
			Password:    cell(row, "密码"),
			Email:       cell(row, "邮箱"),
			Phone:       cell(row, "手机号"),
			Description: cell(row, "描述"),
			Status:      status,
			RoleNames:   cell(row, "角色"),
			Row:         i + 2,
		})
	}

	result, err := ctl.userService.Import(importRows)
	if err != nil {
		utils.Error(c, 500, "获取角色列表失败")
		return
	}
	utils.Success(c, result)
}

// ImportTemplate 下载管理员导入模板
// @Summary 下载管理员导入模板
// @Description 生成仅含表头的 xlsx 模板（用户名/密码/邮箱/手机号/描述/状态/角色）
// @Tags 管理员管理
// @Security BearerAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} binary "Excel 模板"
// @Router /users/import-template [get]
func (ctl *UserController) ImportTemplate(c *gin.Context) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "管理员导入模板"
	f.SetSheetName("Sheet1", sheet)

	sw, _ := f.NewStreamWriter(sheet)
	sw.SetRow("A1", []interface{}{"用户名", "密码", "邮箱", "手机号", "描述", "状态", "角色"})
	sw.Flush()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "G1", headerStyle)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape("管理员导入模板.xlsx"))

	if _, err := f.WriteTo(c.Writer); err != nil {
		utils.ErrorWithStatus(c, http.StatusInternalServerError, 500, "模板生成失败")
	}
}
