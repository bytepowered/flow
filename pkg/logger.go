package flow

import (
	"fmt"
	"github.com/bytepowered/runv"
	"github.com/bytepowered/runv/ext"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"runtime"
	"strings"
)

func init() {
	runv.SetLogger(&logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.JSONFormatter{
			PrettyPrint: false,
			CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
				return ext.LogShortCaller(frame.Func.Name(), frame.Line), ""
			},
		},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: true,
	})
}

func Log() *logrus.Logger {
	return runv.Log()
}

func SetLogger(logger *logrus.Logger) {
	runv.SetLogger(logger)
}

func InitLogger() error {
	v, err := NewLogger()
	if err == nil {
		SetLogger(v)
	}
	return err
}

func NewLogger() (*logrus.Logger, error) {
	viper.SetDefault("app.log.path", "logs")
	viper.SetDefault("app.log.level", "debug")
	viper.SetDefault("app.log.format", "json")
	viper.SetDefault("app.log.caller", true)
	var (
		formatter logrus.Formatter
		fields    = logrus.FieldMap{
			logrus.FieldKeyTime:  "@timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "msg",
			logrus.FieldKeyFunc:  "caller",
		}
		caller = func(frame *runtime.Frame) (function string, file string) {
			return ext.LogShortCaller(frame.Func.Name(), frame.Line), ""
		}
	)
	if strings.EqualFold("json", viper.GetString("app.log.format")) {
		formatter = &logrus.JSONFormatter{
			PrettyPrint:      false,
			FieldMap:         fields,
			CallerPrettyfier: caller,
		}
	} else {
		formatter = &logrus.TextFormatter{
			DisableColors:    true,
			ForceColors:      false,
			CallerPrettyfier: caller,
		}
	}
	level, err := logrus.ParseLevel(viper.GetString("app.log.level"))
	if err != nil {
		return nil, fmt.Errorf("fatal parse log level: %w", err)
	}
	dir := viper.GetString("app.log.path")
	_ = os.Mkdir(dir, os.ModePerm)
	file, err := os.OpenFile(dir+"/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("fatal parse log file: %w", err)
	}
	newLogger := &logrus.Logger{
		Out:          io.MultiWriter(os.Stderr, file),
		Formatter:    formatter,
		Hooks:        make(logrus.LevelHooks),
		Level:        level,
		ExitFunc:     os.Exit,
		ReportCaller: viper.GetBool("app.log.caller"),
	}
	return newLogger, nil
}
