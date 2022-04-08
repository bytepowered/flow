package flow

import (
	"fmt"
	"github.com/bytepowered/runv"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"syscall"
)

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

func run(opts ...EventEngineOption) {
	runv.Add(NewEventEngine(opts...))
	runv.RunV()
}
