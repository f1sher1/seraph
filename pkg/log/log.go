package log

import (
	"fmt"
	"os"
	"path"
	"sync"

	global_config "seraph/app/config"
	"seraph/pkg/contextx"

	"github.com/sirupsen/logrus"
)

var (
	defaultLoggerName = "seraph"
	loggerMu          sync.Mutex
	config            = global_config.Config.LOG
)

func Initialize(format string, timeFormat string) {
	if format != "" {
		config.Format = format
	}
	if timeFormat != "" {
		config.TimestampFormat = timeFormat
	}
	GetLogger(nil, defaultLoggerName)
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func setupLogger() *logrus.Logger {
	formatter := NewLogFormatter()
	if config.TimestampFormat != "" {
		formatter.TimestampFormat = config.TimestampFormat
	}
	if config.Format != "" {
		formatter.OutputFormat = config.Format
	}
	if exists, err := PathExists(config.DirPath); err == nil {
		if !exists {
			os.MkdirAll(config.DirPath, 0770)
		}
	} else {
		panic(err)
	}
	outlog := path.Join(config.DirPath, "seraph.log")

	file, err := os.OpenFile(outlog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err)
	}
	logger := logrus.New()
	logger.SetOutput(file)
	logger.SetLevel(logrus.TraceLevel)
	logger.SetFormatter(formatter)
	return logger
}

func GetLogger(ctx interface{}, name string) *logrus.Entry {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	logger := setupLogger()
	workflow := "-"
	requestId := "-"
	switch t := ctx.(type) {
	case string:
		workflow = t
	case *contextx.Context:
		if t != nil {
			if w, ok := t.GetMap()["workflow"]; ok {
				workflow = w.(string)
			}
			if r, ok := t.GetMap()["requestId"]; ok {
				requestId = r.(string)
			}
		}
	case map[string]interface{}:
		if w, ok := t["workflow"]; ok {
			workflow = w.(string)
		}
		if r, ok := t["requestId"]; ok {
			requestId = r.(string)
		}

	}
	return logger.WithFields(map[string]interface{}{
		"name":      name,
		"requestId": requestId,
		"workflow":  workflow,
	})
}

func Info(ctx interface{}, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Info(args...)
}

func Debug(ctx interface{}, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Debug(args...)
}

func Trace(ctx interface{}, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Trace(args...)
}

func Warn(ctx interface{}, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Warn(args...)
}

func Panic(ctx interface{}, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Panic(args...)
}

func Error(ctx interface{}, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Error(args...)
}

func Infof(ctx interface{}, format string, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Infof(format, args...)
}

func Debugf(ctx interface{}, format string, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Debugf(format, args...)
}

func Tracef(ctx interface{}, format string, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Tracef(format, args...)
}

func Warnf(ctx interface{}, format string, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Warnf(format, args...)
}

func Panicf(ctx interface{}, format string, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Panicf(format, args...)
}

func Errorf(ctx interface{}, format string, args ...interface{}) {
	logger := GetLogger(ctx, defaultLoggerName)
	logger.Errorf(format, args...)
}
