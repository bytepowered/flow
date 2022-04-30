package flow

import (
	"bytes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEngineExecute(t *testing.T) {
	content := `
[engine]
workmode = "single"
[engine.logger]
caller = false
format = "text"
level = "debug"
[engine.logger.text]
force_color = true
pad_level = true
disable_level_truncation = false
disable_timestamp = false
`
	viper.SetConfigType("toml")
	assert.Nil(t, SetupWithConfig(bytes.NewReader([]byte(content))), "setup")
	Register(new(NopInput))
	Register(new(NopOutput))
	Register(new(NopFilter))
	Register(new(NopTransformer))
	Execute()
}
