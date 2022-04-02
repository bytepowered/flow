package output

import (
	"context"
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/sirupsen/logrus"
)

const consoleTag = "console"

var _ flow.Output = new(ConsoleWriter)

type ConsoleWriter struct {
	Level logrus.Level
}

func (c *ConsoleWriter) Tag() string {
	return consoleTag
}

func (c *ConsoleWriter) OnSend(ctx context.Context, events ...flow.Event) {
	for _, evt := range events {
		flow.Log().Logf(c.Level, "tag: %s, time: %s, data: %+v",
			evt.Tag(), evt.Time().String(), evt.Record())
	}
}
