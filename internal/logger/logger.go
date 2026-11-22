package logger

import "go.uber.org/zap"

func New(level string) (*zap.SugaredLogger, error) {
    lvl, err := zap.ParseAtomicLevel(level)
    if err != nil {
        return nil, err
    }

    cfg := zap.NewProductionConfig()
    cfg.Level = lvl

    log, err := cfg.Build()
    if err != nil {
    	return nil, err
    }
    return log.Sugar(), err
}
