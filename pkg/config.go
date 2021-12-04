package flow

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	ConfigKeySource      = "source"
	ConfigKeyFilter      = "filter"
	ConfigKeyTransformer = "transformer"
	ConfigKeyDispatcher  = "dispatcher"
)

type Configurations []map[interface{}]interface{}

func (c Configurations) MarshalLookup(typeid string, outptr interface{}) error {
	for _, m := range c {
		if typeid == cast.ToString(m["typeid"]) {
			decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				TagName:  "toml",
				Metadata: nil,
				Result:   outptr,
			})
			if err != nil {
				return err
			}
			return decoder.Decode(m)
		}
	}
	return nil
}

func (c Configurations) ActiveOf(typeid string) bool {
	for _, m := range c {
		if typeid == cast.ToString(m["typeid"]) {
			return !cast.ToBool(m["disabled"])
		}
	}
	return false
}

var configc = make(map[string][]map[interface{}]interface{}, 4)

type BasicConfig struct {
	TypeId string `toml:"typeid"`
	Tag    string `toml:"tag"`
}

func ConfigurationsOf(typkey string) Configurations {
	cs, ok := configc[typkey]
	if !ok {
		cs = make([]map[interface{}]interface{}, 0)
		_ = viper.UnmarshalKey(typkey, &cs)
		configc[typkey] = cs
	}
	return cs
}
