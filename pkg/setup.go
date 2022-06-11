package flow

import (
	"fmt"
	"github.com/spf13/viper"
	"io"
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

func SetupWithConfig(config io.Reader) error {
	if err := viper.ReadConfig(config); err != nil {
		return fmt.Errorf("read config: %+v", err)
	}
	if err := InitLogger(); err != nil {
		return fmt.Errorf("init logger: %+v", err)
	}
	return nil
}
