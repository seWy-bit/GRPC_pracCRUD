package logger

import (
	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func Init(level string) error {
	var cfg zap.Config

	switch level {
	case "development", "dev", "debug":
		cfg = zap.NewDevelopmentConfig()
	default:
		cfg = zap.NewProductionConfig()
	}

	Logger, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = Logger.Sugar()
	return nil
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
