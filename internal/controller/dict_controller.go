package controller

import (
	"ginproject/internal/model"
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---- DictType Controller ----

type DictTypeController struct {
	dictTypeService *service.DictTypeService
}

func NewDictTypeController(s *service.DictTypeService) *DictTypeController {
	return &DictTypeController{dictTypeService: s}
}

// List 获取字典类型分页列表
// @Summary 获取字典类型分页列表
// @Description 分页查询字典类型列表，支持关键词搜索
// @Tags 数据字典
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词（名称/编码）"
// @Success 200 {object} utils.Response{data=object{list=[]model.DictType,total=int}} "成功"
// @Router /dict-types [get]
func (ctl *DictTypeController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	list, total, err := ctl.dictTypeService.FindPage(page, pageSize, keyword)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": list, "total": total})
}

// Get 获取字典类型详情
// @Summary 获取字典类型详情
// @Description 根据 ID 查询字典类型详情
// @Tags 数据字典
// @Security BearerAuth
// @Produce json
// @Param id path int true "字典类型 ID"
// @Success 200 {object} utils.Response{data=model.DictType} "成功"
// @Failure 200 {object} utils.Response "字典类型不存在"
// @Router /dict-types/{id} [get]
func (ctl *DictTypeController) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	dt, err := ctl.dictTypeService.FindByID(uint(id))
	if err != nil {
		utils.Error(c, 404, "字典类型不存在")
		return
	}
	utils.Success(c, dt)
}

// Create 新建字典类型
// @Summary 新建字典类型
// @Tags 数据字典
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.DictType true "字典类型信息"
// @Success 200 {object} utils.Response "成功"
// @Router /dict-types [post]
func (ctl *DictTypeController) Create(c *gin.Context) {
	var m model.DictType
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.dictTypeService.Create(&m); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Update 编辑字典类型
// @Summary 编辑字典类型
// @Tags 数据字典
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "字典类型 ID"
// @Param body body model.DictType true "字典类型信息"
// @Success 200 {object} utils.Response "成功"
// @Router /dict-types/{id} [put]
func (ctl *DictTypeController) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var m model.DictType
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	m.ID = uint(id)
	if err := ctl.dictTypeService.Update(&m); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Delete 删除字典类型
// @Summary 删除字典类型
// @Description 根据 ID 删除字典类型，存在关联数据项时拒绝删除
// @Tags 数据字典
// @Security BearerAuth
// @Produce json
// @Param id path int true "字典类型 ID"
// @Success 200 {object} utils.Response "成功"
// @Failure 200 {object} utils.Response "存在关联数据项，无法删除"
// @Router /dict-types/{id} [delete]
func (ctl *DictTypeController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.dictTypeService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// ---- DictData Controller ----

type DictDataController struct {
	dictDataService *service.DictDataService
}

func NewDictDataController(s *service.DictDataService) *DictDataController {
	return &DictDataController{dictDataService: s}
}

// List 获取字典数据分页列表
// @Summary 获取字典数据分页列表
// @Description 根据字典类型 ID 分页查询数据项
// @Tags 数据字典
// @Security BearerAuth
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param dict_type_id query int false "字典类型 ID"
// @Success 200 {object} utils.Response{data=object{list=[]model.DictData,total=int}} "成功"
// @Router /dict-data [get]
func (ctl *DictDataController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	dictTypeID, _ := strconv.ParseUint(c.DefaultQuery("dict_type_id", "0"), 10, 64)
	list, total, err := ctl.dictDataService.FindPage(page, pageSize, uint(dictTypeID))
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, gin.H{"list": list, "total": total})
}

// GetByCode 根据字典编码获取所有启用的数据项
// @Summary 根据字典编码获取数据项
// @Tags 数据字典
// @Security BearerAuth
// @Produce json
// @Param code path string true "字典编码（如 gender、user_status）"
// @Success 200 {object} utils.Response{data=[]model.DictData} "成功"
// @Router /dict-data/by-code/{code} [get]
func (ctl *DictDataController) GetByCode(c *gin.Context) {
	code := c.Param("code")
	list, err := ctl.dictDataService.FindByDictTypeCode(code)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, list)
}

// Get 获取字典数据详情
// @Summary 获取字典数据详情
// @Tags 数据字典
// @Security BearerAuth
// @Produce json
// @Param id path int true "字典数据 ID"
// @Success 200 {object} utils.Response{data=model.DictData} "成功"
// @Failure 200 {object} utils.Response "字典数据不存在"
// @Router /dict-data/{id} [get]
func (ctl *DictDataController) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	dd, err := ctl.dictDataService.FindByID(uint(id))
	if err != nil {
		utils.Error(c, 404, "字典数据不存在")
		return
	}
	utils.Success(c, dd)
}

// Create 新建字典数据
// @Summary 新建字典数据
// @Tags 数据字典
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body model.DictData true "字典数据信息"
// @Success 200 {object} utils.Response "成功"
// @Router /dict-data [post]
func (ctl *DictDataController) Create(c *gin.Context) {
	var m model.DictData
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	if err := ctl.dictDataService.Create(&m); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Update 编辑字典数据
// @Summary 编辑字典数据
// @Tags 数据字典
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "字典数据 ID"
// @Param body body model.DictData true "字典数据信息"
// @Success 200 {object} utils.Response "成功"
// @Router /dict-data/{id} [put]
func (ctl *DictDataController) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var m model.DictData
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}
	m.ID = uint(id)
	if err := ctl.dictDataService.Update(&m); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}

// Delete 删除字典数据
// @Summary 删除字典数据
// @Tags 数据字典
// @Security BearerAuth
// @Produce json
// @Param id path int true "字典数据 ID"
// @Success 200 {object} utils.Response "成功"
// @Router /dict-data/{id} [delete]
func (ctl *DictDataController) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := ctl.dictDataService.Delete(uint(id)); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}
	utils.Success(c, nil)
}
