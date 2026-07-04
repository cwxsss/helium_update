package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"path/filepath"
)

var logger *zap.SugaredLogger

func InitLogger(level string) {
	writeSyncer := getLogWriter(level)
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	l := zap.New(core, zap.AddCaller())
	logger = l.Sugar()
}
func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}
func getLogWriter(level string) zapcore.WriteSyncer {
	ws := io.MultiWriter(os.Stdout)
	logPath := getLogFilePath()
	if file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		ws = io.MultiWriter(file, os.Stdout)
	}
	if level == "1" {
		ex, err := os.Executable()
		if err != nil {
			return zapcore.AddSync(ws)
		}
		exPath := filepath.Dir(ex)
		if file, err := os.OpenFile(filepath.Join(exPath, "debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			ws = io.MultiWriter(file, ws)
		}
	}
	return zapcore.AddSync(ws)
}

func getLogFilePath() string {
	appDataDir := os.Getenv("APPDATA")
	if appDataDir == "" {
		appDataDir = os.TempDir()
	}
	logDir := filepath.Join(appDataDir, "helium_updater")
	_ = os.MkdirAll(logDir, os.ModePerm)
	return filepath.Join(logDir, "debug.log")
}
