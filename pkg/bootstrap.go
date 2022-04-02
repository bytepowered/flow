package flow

import (
	"fmt"
	"github.com/bytepowered/runv"
	"github.com/spf13/viper"
)

func SetupConfig() error {
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

func Bootstarp() {
	runv.Add(new(EventEngine))
	runv.RunV()
}
