package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
)

// ImageImporter 镜像导入服务
type ImageImporter struct {
	dbResolver  *dbresolver.DBResolver
	basePath    string // 添加项目根路径字段
	csvFilePath string
}

// NewImageImporter 创建新的镜像导入服务
func NewImageImporter(dbResolver *dbresolver.DBResolver, basePath string, csvFilePath string) *ImageImporter {
	return &ImageImporter{
		dbResolver:  dbResolver,
		basePath:    basePath,
		csvFilePath: csvFilePath,
	}
}

// ImportFromCSV 从CSV文件导入镜像
func (i *ImageImporter) ImportFromCSV(ctx context.Context) error {
	// 打开CSV文件
	file, err := os.Open(i.basePath + "/" + i.csvFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 创建CSV读取器
	reader := csv.NewReader(file)
	// 跳过标题行
	if _, err := reader.Read(); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// 读取并导入数据
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read record: %w", err)
		}

		if err := i.importRecord(ctx, record); err != nil {
			zap.L().Error("Failed to import record",
				zap.Strings("record", record),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// importRecord 导入单条记录
func (i *ImageImporter) importRecord(ctx context.Context, record []string) error {
	if len(record) < 7 {
		return fmt.Errorf("invalid record length: %d", len(record))
	}

	pictureRelativePath := strings.TrimSpace(record[8])
	var pictureFullPath string
	if pictureRelativePath != "" {
		// 使用项目根路径拼接相对路径
		pictureFullPath = filepath.Join(i.basePath, pictureRelativePath)

		// 验证文件是否存在
		if _, err := os.Stat(pictureFullPath); os.IsNotExist(err) {
			return fmt.Errorf("图片文件不存在: %s", pictureFullPath)
		}
	}

	// 解析记录
	image := &model.VMImage{
		Name:            strings.TrimSpace(record[0]),
		DisplayName:     strings.TrimSpace(record[1]),
		OSType:          strings.TrimSpace(record[2]),
		OSVersion:       strings.TrimSpace(record[3]),
		Architecture:    strings.TrimSpace(record[4]),
		ImageURL:        strings.TrimSpace(record[5]),
		DefaultUser:     strings.TrimSpace(record[6]),
		DefaultPassword: strings.TrimSpace(record[7]),
		PictureUrl:      pictureRelativePath,
		Description:     strings.TrimSpace(record[9]),
		Status:          model.ImageStatusAvailable,
		Public:          true, // 系统镜像默认为公共镜像
	}

	// 检查必填字段
	if image.Name == "" || image.OSType == "" || image.OSVersion == "" ||
		image.Architecture == "" || image.ImageURL == "" || image.DefaultUser == "" {
		return fmt.Errorf("missing required fields")
	}

	// 检查镜像是否已存在
	exists, err := dao.CheckImageExists(ctx, i.dbResolver, image.Name)
	if err != nil {
		return fmt.Errorf("failed to check image existence: %w", err)
	}

	if exists {
		// 更新现有镜像
		if err := dao.UpdateImageByName(ctx, i.dbResolver, image); err != nil {
			return fmt.Errorf("failed to update image: %w", err)
		}
	} else {
		// 创建新镜像
		if err := dao.InsertImage(ctx, i.dbResolver, image); err != nil {
			return fmt.Errorf("failed to insert image: %w", err)
		}
	}

	return nil
}
