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
	viper.SetDefault("engine.logger.path", "logs")
	viper.SetDefault("engine.logger.level", "debug")
	viper.SetDefault("engine.logger.format", "json")
	viper.SetDefault("engine.logger.caller", true)
	var (
		formatter logrus.Formatter
		fields    = logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "lv",
			logrus.FieldKeyMsg:   "msg",
			logrus.FieldKeyFunc:  "caller",
		}
		caller = func(frame *runtime.Frame) (function string, file string) {
			return ext.LogShortCaller(frame.Func.Name(), frame.Line), ""
		}
	)
	switch strings.ToLower(viper.GetString("engine.logger.format")) {
	case "json":
		formatter = &logrus.JSONFormatter{
			PrettyPrint:      viper.GetBool("engine.logger.json.pretty"),
			DisableTimestamp: viper.GetBool("engine.logger.json.disable_timestamp"),
			FieldMap:         fields,
			CallerPrettyfier: caller,
		}
	case "text":
		formatter = &logrus.TextFormatter{
			DisableColors:          viper.GetBool("engine.logger.text.disable_color"),
			ForceColors:            viper.GetBool("engine.logger.text.force_color"),
			ForceQuote:             viper.GetBool("engine.logger.text.force_quote"),
			PadLevelText:           viper.GetBool("engine.logger.text.pad_level"),
			DisableLevelTruncation: viper.GetBool("engine.logger.text.disable_level_truncation"),
			DisableSorting:         viper.GetBool("engine.logger.text.disable_sorting"),
			DisableTimestamp:       viper.GetBool("engine.logger.text.disable_timestamp"),
			CallerPrettyfier:       caller,
		}
	}
	level, err := logrus.ParseLevel(viper.GetString("engine.logger.level"))
	if err != nil {
		return nil, fmt.Errorf("fatal parse log level: %w", err)
	}
	dir := viper.GetString("engine.logger.path")
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
		ReportCaller: viper.GetBool("engine.logger.caller"),
	}
	return newLogger, nil
}
