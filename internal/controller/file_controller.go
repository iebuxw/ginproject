package controller

import (
	"ginproject/internal/service"
	"ginproject/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FileController struct {
	fileService *service.FileService
}

func NewFileController(fileService *service.FileService) *FileController {
	return &FileController{fileService: fileService}
}

func (c *FileController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	name := ctx.Query("name")

	list, total, err := c.fileService.List(page, pageSize, name)
	if err != nil {
		utils.Error(ctx, 500, "查询失败")
		return
	}

	utils.Success(ctx, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *FileController) Upload(ctx *gin.Context) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		utils.Error(ctx, 400, "请选择文件")
		return
	}
	defer file.Close()

	userID, _ := ctx.Get("user_id")
	uploaderID, _ := userID.(uint)
	username, _ := ctx.Get("username")
	uploaderName, _ := username.(string)

	record, err := c.fileService.Upload(header.Filename, header.Size, file, int64(uploaderID), uploaderName)
	if err != nil {
		utils.Error(ctx, 400, err.Error())
		return
	}

	utils.Success(ctx, record)
}

func (c *FileController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "参数错误")
		return
	}

	if err := c.fileService.Delete(id); err != nil {
		utils.Error(ctx, 500, err.Error())
		return
	}

	utils.Success(ctx, nil)
}

func (c *FileController) Download(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		utils.Error(ctx, 400, "参数错误")
		return
	}

	file, err := c.fileService.GetByID(id)
	if err != nil {
		utils.Error(ctx, 404, "文件不存在")
		return
	}

	ctx.FileAttachment(c.fileService.GetFilePath(file.StoredName), file.OriginalName)
}
