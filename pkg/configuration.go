package flow

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	ConfigRootSource      = "source"
	ConfigRootFilter      = "filter"
	ConfigRootTransformer = "transformer"
	ConfigRootOutput      = "output"

	ConfigKeyTypeId   = "type-id"
	ConfigKeyDisabled = "disabled"
)

const configTagName = "toml"

// BasedConfiguration 基础配置必要的字段
type BasedConfiguration struct {
	Tag         string `toml:"tag"`
	Description string `toml:"description"`
}

// Configuration 不定字段配置
type Configuration map[string]interface{}

func (c Configuration) LookupE(key string) (interface{}, bool) {
	v, ok := c[key]
	return v, ok
}

func (c Configuration) Lookup(key string) interface{} {
	return c[key]
}

func (c Configuration) StringOf(key string) string {
	return cast.ToString(c[key])
}

func (c Configuration) SliceOf(key string) []string {
	return cast.ToStringSlice(c[key])
}

func (c Configuration) BoolOf(key string) bool {
	return cast.ToBool(c[key])
}

// Configurations 某些类型的配置列表
type Configurations []Configuration

func (c Configurations) Lookup(typeid string, outptr interface{}) (bool, error) {
	for _, m := range c {
		if typeid == m.StringOf(ConfigKeyTypeId) {
			decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				TagName:  configTagName,
				Squash:   true,
				Metadata: nil,
				Result:   outptr,
			})
			if err != nil {
				return true, err
			}
			return true, decoder.Decode(m)
		}
	}
	return false, nil
}

func (c Configurations) ActiveOf(typeid string) bool {
	for _, m := range c {
		if typeid == m.StringOf(ConfigKeyTypeId) {
			return m.BoolOf(ConfigKeyDisabled) || m.BoolOf("disable")
		}
	}
	return false
}

var _ccached = make(map[string][]Configuration, 4)

func RootConfigOf(root string) Configurations {
	cs, ok := _ccached[root]
	if !ok {
		cs = make([]Configuration, 0)
		_ = UnmarshalKey(root, &cs)
		_ccached[root] = cs
	}
	return cs
}

func UnmarshalKey(key string, valptr interface{}) error {
	return viper.UnmarshalKey(key, valptr, withTomlTag(configTagName), withSlash(true))
}

func withTomlTag(tag string) func(config *mapstructure.DecoderConfig) {
	return func(c *mapstructure.DecoderConfig) {
		c.TagName = tag
	}
}

func withSlash(squash bool) func(config *mapstructure.DecoderConfig) {
	return func(c *mapstructure.DecoderConfig) {
		c.Squash = squash
	}
}
