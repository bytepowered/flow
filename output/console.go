package output

import (
	"context"
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const consoleTag = "console"

var _ flow.Output = new(ConsoleWriter)
var _ runv.Initable = new(ConsoleWriter)

type ConsoleWriter struct {
	Level      logrus.Level
	ShowHeader bool
}

func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{
		Level:      logrus.InfoLevel,
		ShowHeader: false,
	}
}

func (c *ConsoleWriter) OnInit() error {
	viper.SetDefault("console.level", "info")
	viper.SetDefault("console.show-header", false)
	lv, err := logrus.ParseLevel(viper.GetString("console.level"))
	if err != nil {
		return err
	}
	c.Level = lv
	c.ShowHeader = viper.GetBool("console.show-header")
	flow.Log().Infof("CONSOLE: options, level=%s, show-header:%v", lv, c.ShowHeader)
	return nil
}

func (c *ConsoleWriter) Tag() string {
	return consoleTag
}

func (c *ConsoleWriter) OnSend(ctx context.Context, events ...flow.Event) {
	for _, evt := range events {
		if c.ShowHeader {
			flow.Log().WithField("header", evt.Header()).Logf(c.Level, "data: %s", evt.Record())
		} else {
			flow.Log().Logf(c.Level, "data: %s", evt.Record())
		}
	}
}
