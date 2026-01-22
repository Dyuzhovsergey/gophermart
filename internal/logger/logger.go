// Package logger provides zap logger initialization.
package logger

import "go.uber.org/zap"

func Init() (*zap.Logger, error) {
	return ezap.NewDevelopment()
}
