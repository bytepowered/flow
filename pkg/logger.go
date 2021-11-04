package flow

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"runtime"
	"strings"
)

var (
	_logger *logrus.Logger
)

func Log() *logrus.Logger {
	return _logger
}

func InitLogger() error {
	viper.SetDefault("app.log.level", "info")
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
			return shortCaller(frame.Func.Name()), ""
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
		return fmt.Errorf("fatal parse log level: %w", err)
	}
	const dir = "logs"
	_ = os.Mkdir(dir, os.ModePerm)
	file, err := os.OpenFile(dir+"/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("fatal parse log file: %w", err)
	}

	_logger = &logrus.Logger{
		Out:          io.MultiWriter(os.Stdout, file),
		Formatter:    formatter,
		Hooks:        make(logrus.LevelHooks),
		Level:        level,
		ExitFunc:     os.Exit,
		ReportCaller: viper.GetBool("app.log.caller"),
	}
	return nil
}

func shortCaller(caller string) string {
	// cmschina.com.cn/msd/pkg/impl/szfiu.(*WebsocketMarketAdapter).OnInit
	// cmschina.com.cn/msd/pkg/impl/szfiu.(*WebsocketMarketAdapter).OnStart
	sbytes := []byte(caller)
	idx := bytes.LastIndexByte(sbytes, '(')
	if idx <= 0 {
		return caller
	}
	return string(sbytes[idx:])
}
