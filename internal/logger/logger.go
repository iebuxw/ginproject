package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var L *zap.Logger

// LogConfig 日志配置
type LogConfig struct {
	Level      string // debug, info, warn, error
	FilePath   string // 日志文件路径，空则仅输出控制台
	MaxSize    int    // 单文件最大 MB
	MaxBackups int    // 保留旧文件个数
	MaxDays    int    // 保留天数
}

// Init 初始化全局日志器；应在 config.Load() 之后、业务启动之前调用
func Init(cfg LogConfig) {
	level := parseLevel(cfg.Level)

	// 编码器配置：控制台用彩色文本，文件用 JSON
	consoleEnc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	fileEncCfg := zap.NewProductionEncoderConfig()
	fileEncCfg.TimeKey = "ts"
	fileEncCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	fileEnc := zapcore.NewJSONEncoder(fileEncCfg)

	// 控制台输出
	consoleCore := zapcore.NewCore(consoleEnc, zapcore.AddSync(os.Stdout), level)

	cores := []zapcore.Core{consoleCore}

	// 文件输出（配置了路径才启用）
	if cfg.FilePath != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxDays,
			Compress:   true,
		}
		fileCore := zapcore.NewCore(fileEnc, zapcore.AddSync(fileWriter), level)
		cores = append(cores, fileCore)
	}

	L = zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(0))
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// 便捷方法，直接代理到全局 L

func Debug(msg string, fields ...zap.Field) { L.Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)  { L.Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { L.Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { L.Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { L.Fatal(msg, fields...) }

// Sync 刷盘，进程退出前调用
func Sync() { _ = L.Sync() }
