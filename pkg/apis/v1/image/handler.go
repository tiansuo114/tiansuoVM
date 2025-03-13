package image

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tiansuoVM/pkg/apis/v1/logs"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/server/encoding"
	"tiansuoVM/pkg/server/errutil"
	"tiansuoVM/pkg/server/request"
	"tiansuoVM/pkg/token"
	"tiansuoVM/pkg/types"
)

type handler struct {
	dbResolver *dbresolver.DBResolver
}

func newHandler(dbResolver *dbresolver.DBResolver) *handler {
	return &handler{dbResolver: dbResolver}
}

// listImages 列出镜像
func (h *handler) listImages(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req listImageReq
	if err := c.ShouldBindQuery(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 处理分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	var (
		images []*model.VMImage
		total  int64
		err    error
	)

	switch {
	case req.Public:
		images, err = dao.ListPublicImages(ctx, h.dbResolver)
	case req.Type != "":
		images, total, err = dao.ListImagesByOSType(ctx, h.dbResolver, req.Type, req.Pagination)
	default:
		images, total, err = dao.ListImages(ctx, h.dbResolver, req.Pagination)
	}

	if err != nil {
		zap.L().Error("获取镜像列表失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 转换为响应结构体
	imageInfos := make([]imageInfo, 0)
	for _, image := range images {
		info := imageInfo{
			ID:           image.ID,
			Name:         image.Name,
			DisplayName:  image.DisplayName,
			OSType:       image.OSType,
			OSVersion:    image.OSVersion,
			Architecture: image.Architecture,
			ImageURL:     image.ImageURL,
			Public:       image.Public,
			DefaultUser:  image.DefaultUser,
			Description:  image.Description,
			Status:       string(image.Status),
			CreatedAt:    image.CreatedAt,
			Creator:      image.Creator,
			UpdatedAt:    image.UpdatedAt,
			Updater:      image.Updater,
		}
		imageInfos = append(imageInfos, info)
	}

	encoding.HandleSuccessList(c, total, imageInfos)
}

// getImage 获取镜像详情
func (h *handler) getImage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	image, err := dao.GetImageByID(ctx, h.dbResolver, id)
	if err != nil {
		zap.L().Error("获取镜像信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	if image == nil {
		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "镜像不存在"))
		return
	}

	// 转换为响应结构体
	imageInfo := imageInfo{
		ID:           image.ID,
		Name:         image.Name,
		DisplayName:  image.DisplayName,
		OSType:       image.OSType,
		OSVersion:    image.OSVersion,
		Architecture: image.Architecture,
		ImageURL:     image.ImageURL,
		Public:       image.Public,
		DefaultUser:  image.DefaultUser,
		Description:  image.Description,
		Status:       string(image.Status),
		CreatedAt:    image.CreatedAt,
		Creator:      image.Creator,
		UpdatedAt:    image.UpdatedAt,
		Updater:      image.Updater,
	}

	encoding.HandleSuccess(c, imageInfo)
}

// createImage 创建镜像
func (h *handler) createImage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req createImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 检查镜像名称是否已存在
	exists, err := dao.CheckImageExists(ctx, h.dbResolver, req.Name)
	if err != nil {
		zap.L().Error("检查镜像存在性失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if exists {
		encoding.HandleError(c, errutil.NewError(http.StatusConflict, "镜像名称已存在"))
		return
	}

	// 获取操作者信息
	operator := token.GetUIDFromCtx(c)

	newImage := &model.VMImage{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		OSType:       req.OSType,
		OSVersion:    req.OSVersion,
		Architecture: req.Architecture,
		ImageURL:     req.ImageURL,
		Public:       req.Public,
		DefaultUser:  req.DefaultUser,
		Description:  req.Description,
		Status:       model.ImageStatusAvailable,
		Creator:      operator,
		Updater:      operator,
	}

	if err := dao.InsertImage(ctx, h.dbResolver, newImage); err != nil {
		zap.L().Error("创建镜像失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	imageInfo := imageInfo{
		ID:           newImage.ID,
		Name:         newImage.Name,
		DisplayName:  newImage.DisplayName,
		OSType:       newImage.OSType,
		OSVersion:    newImage.OSVersion,
		Architecture: newImage.Architecture,
		ImageURL:     newImage.ImageURL,
		Public:       newImage.Public,
		DefaultUser:  newImage.DefaultUser,
		Description:  newImage.Description,
		Status:       string(newImage.Status),
		CreatedAt:    newImage.CreatedAt,
		Creator:      newImage.Creator,
		UpdatedAt:    newImage.UpdatedAt,
		Updater:      newImage.Updater,
	}

	encoding.HandleSuccess(c, imageInfo)

	// 记录操作日志
	logs.EventLogChannel <- &model.EventLog{
		ResourceType: model.ResourceTypeImage,
		ResourceUID:  strconv.FormatInt(newImage.ID, 10),
		EventType:    model.EventTypeCreation,
		Creator:      operator,
		CreatedAt:    time.Now().UnixMilli(),
	}
}

// updateImage 更新镜像
func (h *handler) updateImage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	var req updateImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 获取现有镜像
	existing, err := dao.GetImageByID(ctx, h.dbResolver, id)
	if err != nil {
		zap.L().Error("获取镜像信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if existing == nil {
		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "镜像不存在"))
		return
	}

	// 更新字段
	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Status != "" {
		existing.Status = model.ImageStatus(req.Status)
	}
	if req.Public != nil {
		existing.Public = *req.Public
	}

	existing.Updater = token.GetUIDFromCtx(c)
	existing.UpdatedAt = time.Now().UnixMilli()

	if err := dao.UpdateImage(ctx, h.dbResolver, existing); err != nil {
		zap.L().Error("更新镜像失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 记录操作日志
	logs.EventLogChannel <- &model.EventLog{
		ResourceType: model.ResourceTypeImage,
		ResourceUID:  strconv.FormatInt(existing.ID, 10),
		EventType:    model.EventTypeUpdate,
		Creator:      existing.Updater,
		CreatedAt:    time.Now().UnixMilli(),
	}

	encoding.HandleSuccess(c, existing)
}

//// deleteImage 删除镜像
//func (h *handler) deleteImage(c *gin.Context) {
//	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
//	defer cancel()
//
//	idStr := c.Param("id")
//	id, err := strconv.ParseInt(idStr, 10, 64)
//	if err != nil {
//		encoding.HandleError(c, errutil.ErrIllegalParameter)
//		return
//	}
//
//	// 检查镜像是否存在
//	existing, err := dao.GetImageByID(ctx, h.dbResolver, id)
//	if err != nil {
//		zap.L().Error("获取镜像信息失败", zap.Error(err))
//		encoding.HandleError(c, errutil.ErrInternalServer)
//		return
//	}
//	if existing == nil {
//		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "镜像不存在"))
//		return
//	}
//
//	// 检查是否有关联的VM
//	count, err := dao.CountVMsByImage(ctx, h.dbResolver, id)
//	if err != nil {
//		zap.L().Error("检查镜像使用情况失败", zap.Error(err))
//		encoding.HandleError(c, errutil.ErrInternalServer)
//		return
//	}
//	if count > 0 {
//		encoding.HandleError(c, errutil.NewError(http.StatusBadRequest, "存在使用该镜像的虚拟机，无法删除"))
//		return
//	}
//
//	if err := dao.DeleteImage(ctx, h.dbResolver, id); err != nil {
//		zap.L().Error("删除镜像失败", zap.Error(err))
//		encoding.HandleError(c, errutil.ErrInternalServer)
//		return
//	}
//
//	// 记录操作日志
//	logs.EventLogChannel <- &model.EventLog{
//		ResourceType: model.ResourceTypeImage,
//		ResourceUID:  strconv.FormatInt(id, 10),
//		EventType:    model.EventTypeDeletion,
//		Creator:      token.GetUIDFromCtx(c),
//		CreatedAt:    time.Now().UnixMilli(),
//	}
//
//	encoding.HandleSuccess(c, nil)
//}
