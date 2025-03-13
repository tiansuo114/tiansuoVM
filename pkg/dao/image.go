package dao

import (
	"context"
	"errors"
	"tiansuoVM/pkg/server/request"
	"time"

	"gorm.io/gorm"

	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
)

// InsertImage creates a new virtual machine image record in the database.
func InsertImage(ctx context.Context, dbResolver *dbresolver.DBResolver, image *model.VMImage) error {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	return db.WithContext(ctx).Create(image).Error
}

// GetImageByID retrieves a virtual machine image record by its ID.
func GetImageByID(ctx context.Context, dbResolver *dbresolver.DBResolver, id int64) (*model.VMImage, error) {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	image := &model.VMImage{}
	err := db.WithContext(ctx).Where("id = ?", id).First(image).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return image, nil
}

// GetImageByName retrieves a virtual machine image record by its name.
func GetImageByName(ctx context.Context, dbResolver *dbresolver.DBResolver, name string) (*model.VMImage, error) {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	image := &model.VMImage{}
	err := db.WithContext(ctx).Where("name = ?", name).First(image).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return image, nil
}

// UpdateImage updates the virtual machine image record.
func UpdateImage(ctx context.Context, dbResolver *dbresolver.DBResolver, image *model.VMImage) error {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	image.UpdatedAt = time.Now().UnixMilli()
	return db.WithContext(ctx).Save(image).Error
}

// UpdateImageByName updates the virtual machine image record by its name.
func UpdateImageByName(ctx context.Context, dbResolver *dbresolver.DBResolver, image *model.VMImage) error {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	image.UpdatedAt = time.Now().UnixMilli()
	return db.WithContext(ctx).Where("name = ?", image.Name).Updates(image).Error
}

// DeleteImage deletes a virtual machine image record.
func DeleteImage(ctx context.Context, dbResolver *dbresolver.DBResolver, id int64) error {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	return db.WithContext(ctx).Delete(&model.VMImage{ID: id}).Error
}

// ListImages retrieves all virtual machine image records from the database.
func ListImages(ctx context.Context, dbResolver *dbresolver.DBResolver, pagination request.Pagination) ([]*model.VMImage, int64, error) {
	db := dbResolver.GetDB().WithContext(ctx).Model(&model.VMImage{})
	var images []*model.VMImage
	var total int64

	err := db.WithContext(ctx).Count(&total).Error
	if err != nil {
		return nil, total, err
	}

	err = pagination.MakeSQL(db).WithContext(ctx).Find(&images).Error
	if err != nil {
		return images, total, err
	}
	return images, total, nil
}

func ListImagesWithNoPagination(ctx context.Context, dbResolver *dbresolver.DBResolver) ([]*model.VMImage, error) {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	var images []*model.VMImage
	err := db.WithContext(ctx).Find(&images).Error
	if err != nil {
		return nil, err
	}
	return images, nil
}

// ListPublicImages retrieves all public virtual machine image records from the database.
func ListPublicImages(ctx context.Context, dbResolver *dbresolver.DBResolver) ([]*model.VMImage, error) {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	var images []*model.VMImage
	err := db.WithContext(ctx).Where("public = ?", true).Find(&images).Error
	if err != nil {
		return nil, err
	}
	return images, nil
}

// ListImagesByOSType retrieves all virtual machine image records with a specific operating system type.
func ListImagesByOSType(ctx context.Context, dbResolver *dbresolver.DBResolver, osType string, pagination request.Pagination) ([]*model.VMImage, int64, error) {
	db := dbResolver.GetDB().WithContext(ctx).Model(&model.VMImage{})
	var images []*model.VMImage
	db = db.Where("os_type = ?", osType)

	var count int64
	err := db.WithContext(context.Background()).Count(&count).Error
	if err != nil {
		return images, 0, err
	}

	err = db.WithContext(ctx).Where("os_type = ?", osType).Find(&images).Error
	return images, count, err
}

// ListImagesByStatus retrieves all virtual machine image records with a specific status.
func ListImagesByStatus(ctx context.Context, dbResolver *dbresolver.DBResolver, status model.ImageStatus, pagination request.Pagination) ([]model.VMImage, int64, error) {
	db := dbResolver.GetDB().WithContext(ctx).Model(&model.VMImage{})
	var images []model.VMImage

	db = db.Where("status = ?", status)

	var count int64
	err := db.WithContext(context.Background()).Count(&count).Error
	if err != nil {
		return images, 0, err
	}

	err = pagination.MakeSQL(db).Find(&images).Error
	return images, count, err
}

// CountImages counts the number of virtual machine image records.
func CountImages(ctx context.Context, dbResolver *dbresolver.DBResolver) (int64, error) {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	var count int64
	err := db.WithContext(ctx).Model(&model.VMImage{}).Count(&count).Error
	return count, err
}

// CheckImageExists checks if a virtual machine image record exists by its name.
func CheckImageExists(ctx context.Context, dbResolver *dbresolver.DBResolver, name string) (bool, error) {
	db := dbResolver.GetDB().Model(&model.VMImage{})
	var count int64
	err := db.WithContext(ctx).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// InsertImageOperationLog creates a new image operation log record in the database.
func InsertImageOperationLog(ctx context.Context, dbResolver *dbresolver.DBResolver, log *model.ImageOperationLog) error {
	db := dbResolver.GetDB().Model(&model.ImageOperationLog{})
	return db.WithContext(ctx).Create(log).Error
}

// ListImageOperationLogs retrieves all image operation logs for a specific image from the database.
func ListImageOperationLogs(ctx context.Context, dbResolver *dbresolver.DBResolver, imageID int64) ([]*model.ImageOperationLog, error) {
	db := dbResolver.GetDB().Model(&model.ImageOperationLog{})
	var logs []*model.ImageOperationLog
	err := db.WithContext(ctx).Where("image_id = ?", imageID).Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}
