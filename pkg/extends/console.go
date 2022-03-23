package extends

import (
	"context"
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/sirupsen/logrus"
)

var _ flow.Output = new(ConsoleWriter)

type ConsoleWriter struct {
	Level logrus.Level
}

func (c *ConsoleWriter) Tag() string {
	return "console"
}

func (c *ConsoleWriter) OnSend(ctx context.Context, events ...flow.Event) {
	console := flow.Log().WithField("source", c.Tag())
	console.Level = c.Level
	for _, evt := range events {
		console.Infof("tag: %s, kind: %s, time: %s, data: %+v",
			evt.Tag(), evt.Kind().String(), evt.Time().String(), evt.Record())
	}
}
