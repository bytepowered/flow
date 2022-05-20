package flow

import (
	"fmt"
	"github.com/bytepowered/runv"
	"github.com/spf13/viper"
	"io"
	"os"
	"os/signal"
	"syscall"
)

var _enx *EventEngine

func Setup() error {
	SetConfigDefaults()
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %+v", err)
	}
	if err := InitLogger(); err != nil {
		return fmt.Errorf("init logger: %+v", err)
	}
	return nil
}

func SetupWithConfig(config io.Reader) error {
	if err := viper.ReadConfig(config); err != nil {
		return fmt.Errorf("read config: %+v", err)
	}
	if err := InitLogger(); err != nil {
		return fmt.Errorf("init logger: %+v", err)
	}
	return nil
}

func Register(obj interface{}) {
	runv.Add(obj)
}

func Bootstrap(opts ...EventEngineOption) {
	runv.SetSignals(func() <-chan os.Signal {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		return sig
	})
	run(opts...)
}

func Execute(opts ...EventEngineOption) {
	runv.SetSignals(func() <-chan os.Signal {
		sig := make(chan os.Signal, 1)
		sig <- syscall.SIGTERM
		return sig
	})
	run(opts...)
}

func Engine() *EventEngine {
	return _enx
}

func run(opts ...EventEngineOption) {
	_enx = NewEventEngine(opts...)
	for _, comp := range runv.Objects() {
		if i, ok := comp.(Input); ok {
			_enx.AddInput(i)
		} else if f, ok := comp.(Filter); ok {
			_enx.AddFilter(f)
		} else if t, ok := comp.(Transformer); ok {
			_enx.AddTransformer(t)
		} else if o, ok := comp.(Output); ok {
			_enx.AddOutput(o)
		}
	}
	runv.Add(_enx)
	runv.RunV()
}
