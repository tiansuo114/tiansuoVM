package main

import (
	"go.uber.org/zap"
	"tiansuoVM/cmd/console/app"
)

func main() {
	cmd := app.NewAPIServerCommand()
	if err := cmd.Execute(); err != nil {
		zap.S().Fatal(err)
	}
}
