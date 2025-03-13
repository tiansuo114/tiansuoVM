package logs

import (
	"context"
	"go.uber.org/zap"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
)

var UserOperatorLogChannel = make(chan *model.UserOperatorLog, 100)

var EventLogChannel = make(chan *model.EventLog, 100)

func startUserOperatorLogListener(dbResolver *dbresolver.DBResolver) {
	go func() {
		for log := range UserOperatorLogChannel {
			// Insert the log into the database
			if err := dao.InsertUserOperatorLogByModel(context.Background(), dbResolver, log); err != nil {
				zap.L().Error("failed to insert user operator log", zap.Error(err))
			}
		}
	}()
}

func startEventLogListener(dbResolver *dbresolver.DBResolver) {
	go func() {
		for log := range EventLogChannel {
			// Insert the log into the database
			if err := dao.InsertEventLogByModel(context.Background(), dbResolver, log); err != nil {
				zap.L().Error("failed to insert event log", zap.Error(err))
			}
		}
	}()
}
