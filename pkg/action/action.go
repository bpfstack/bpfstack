package action

import (
	"github.com/bpfstack/bpfstack/pkg/logger"
)

// ActionInterface defines the interface for all actions
type ActionInterface interface {
	Name() string
	Init() error
	Start() error
	Stop() error
}

// BaseAction provides common functionality for all actions
type BaseAction struct {
	ActionName string
	Logger     *logger.Logger
}

// NewBaseAction creates a new base action with logger
func NewBaseAction(name string) *BaseAction {
	return &BaseAction{
		ActionName: name,
		Logger:     logger.New(name),
	}
}

// Name returns the action name
func (b *BaseAction) Name() string {
	return b.ActionName
}

// LogInfo logs an info message
func (b *BaseAction) LogInfo(msg string, fields ...logger.Fields) {
	b.Logger.Info(msg, fields...)
}

// LogDebug logs a debug message
func (b *BaseAction) LogDebug(msg string, fields ...logger.Fields) {
	b.Logger.Debug(msg, fields...)
}

// LogWarn logs a warning message
func (b *BaseAction) LogWarn(msg string, fields ...logger.Fields) {
	b.Logger.Warn(msg, fields...)
}

// LogError logs an error message
func (b *BaseAction) LogError(msg string, fields ...logger.Fields) {
	b.Logger.Error(msg, fields...)
}
