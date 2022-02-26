package flow

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"strings"
)

const (
	ConfigTagName = "toml"
)

// BasedConfiguration 基础配置必要的字段
type BasedConfiguration struct {
	Tag         string `toml:"tag"`
	Description string `toml:"description"`
}

func SetConfigDefaults() {
	viper.KeyDelimiter(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetConfigName("application")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./conf.d")
}

func UnmarshalConfigKey(key string, structptr interface{}) error {
	return UnmarshalConfigWith(key, ConfigTagName, structptr)
}

func UnmarshalConfigWith(key, tagName string, structptr interface{}) error {
	return viper.UnmarshalKey(key, structptr, withConfigTagName(tagName), withConfigSlash(true))
}

func withConfigTagName(tag string) func(config *mapstructure.DecoderConfig) {
	return func(c *mapstructure.DecoderConfig) {
		c.TagName = tag
	}
}

func withConfigSlash(squash bool) func(config *mapstructure.DecoderConfig) {
	return func(c *mapstructure.DecoderConfig) {
		c.Squash = squash
	}
}
