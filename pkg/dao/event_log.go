package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"

	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
)

// InsertEventLog creates a new event log entry in the database.
func InsertEventLog(ctx context.Context, dbResolver *dbresolver.DBResolver, eventType model.EventType, operation string) (*model.EventLog, error) {
	db := dbResolver.GetDB()
	return InsertEventLogWithDB(ctx, db, eventType, operation)
}

func InsertEventLogWithDB(ctx context.Context, db *gorm.DB, eventType model.EventType, operation string) (*model.EventLog, error) {
	// Create a new EventLog record
	eventLog := model.EventLog{
		EventType: eventType,
		Operation: operation,
	}

	// Insert the event log record into the database
	err := db.WithContext(ctx).Create(&eventLog).Error
	return &eventLog, err
}

func InsertEventLogByModel(ctx context.Context, dbResolver *dbresolver.DBResolver, eventLog *model.EventLog) error {
	db := dbResolver.GetDB()
	return InsertEventLogByModelWithDB(ctx, db, eventLog)
}

func InsertEventLogByModelWithDB(ctx context.Context, db *gorm.DB, eventLog *model.EventLog) error {
	err := db.WithContext(ctx).Create(&eventLog).Error
	return err
}

// GetEventLogByID retrieves an event log entry by its ID.
func GetEventLogByID(ctx context.Context, dbResolver *dbresolver.DBResolver, id uint) (bool, *model.EventLog, error) {
	db := dbResolver.GetDB()
	return GetEventLogByIDWithDB(ctx, db, id)
}

func GetEventLogByIDWithDB(ctx context.Context, db *gorm.DB, id uint) (bool, *model.EventLog, error) {
	eventLog := model.EventLog{}
	err := db.WithContext(ctx).Where("id = ?", id).First(&eventLog).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &eventLog, nil
}

// ListEventLogs retrieves all event logs from the database.
func ListEventLogs(ctx context.Context, dbResolver *dbresolver.DBResolver) ([]model.EventLog, error) {
	db := dbResolver.GetDB()
	var eventLogs []model.EventLog
	err := db.WithContext(ctx).Find(&eventLogs).Error
	return eventLogs, err
}

// ListEventLogsByType retrieves event logs by a specific event type.
func ListEventLogsByType(ctx context.Context, dbResolver *dbresolver.DBResolver, eventType model.EventType) ([]model.EventLog, error) {
	db := dbResolver.GetDB()
	var eventLogs []model.EventLog
	err := db.WithContext(ctx).Where("event_type = ?", eventType).Find(&eventLogs).Error
	return eventLogs, err
}

// ListEventLogsByCreator retrieves event logs by the user who triggered the event.
func ListEventLogsByCreator(ctx context.Context, dbResolver *dbresolver.DBResolver, creator string) ([]model.EventLog, error) {
	db := dbResolver.GetDB()
	var eventLogs []model.EventLog
	err := db.WithContext(ctx).Where("creator = ?", creator).Find(&eventLogs).Error
	return eventLogs, err
}

// DeleteEventLogByID deletes an event log entry by its ID.
func DeleteEventLogByID(ctx context.Context, dbResolver *dbresolver.DBResolver, id uint) error {
	db := dbResolver.GetDB()
	return db.WithContext(ctx).Where("id = ?", id).Delete(&model.EventLog{}).Error
}

// UpdateEventLogByID updates the details of an event log entry by its ID.
func UpdateEventLogByID(ctx context.Context, dbResolver *dbresolver.DBResolver, id uint, updates map[string]interface{}) error {
	db := dbResolver.GetDB()

	// Update the event log entry
	updates["updated_at"] = time.Now().UnixMilli()
	return db.WithContext(ctx).Model(&model.EventLog{}).Where("id = ?", id).Updates(updates).Error
}
