package flow

import (
	"bytes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestUnmarshalConfigKey(t *testing.T) {
	type Config struct {
		Foo time.Duration `json:"foo" toml:"foo"`
		Bar string        `json:"bar" toml:"bar"`
		Val uint          `json:"val" toml:"val"`
	}
	content := `
[a.b.c.config]
foo = "1h"
bar = "bar"
val = 123
`
	viper.SetConfigType("toml")
	if err := viper.ReadConfig(bytes.NewReader([]byte(content))); err != nil {
		assert.Fail(t, "error: "+err.Error())
	}
	for _, tag := range []string{"json", "toml"} {
		c := Config{}
		assert.Nil(t, UnmarshalConfigWith("a.b.c.config", tag, &c))
		assert.Equal(t, "1h0m0s", c.Foo.String())
		assert.Equal(t, "bar", c.Bar)
		assert.Equal(t, uint(123), c.Val)
	}

}
