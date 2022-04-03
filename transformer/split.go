package transformer

import (
	"bytes"
	"fmt"
	"github.com/bytepowered/flow/v3/extends/events"
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/bytepowered/runv"
	"github.com/spf13/viper"
	"strings"
)

const splitTag = "split"

var _ flow.Transformer = new(SplitTransformer)
var _ runv.Initable = new(SplitTransformer)

func NewSplitTransformer() *SplitTransformer {
	return &SplitTransformer{
		separator: ",",
		subsize:   0,
	}
}

type SplitTransformer struct {
	separator string
	subsize   uint
}

func (c *SplitTransformer) OnInit() error {
	with := func(prefix string) {
		viper.SetDefault(prefix+"separator", ",")
		c.separator = viper.GetString(prefix + "separator")
		c.subsize = viper.GetUint(prefix + "subsize")
	}
	with(c.Tag() + ".")
	return nil
}

func (c *SplitTransformer) Tag() string {
	return splitTag
}

func (c *SplitTransformer) DoTransform(ctx flow.StateContext, in []flow.Event) (out []flow.Event, err error) {
	subsize := int(c.subsize)
	for _, evt := range in {
		data := evt.Record()
		var fields []string
		switch data.(type) {
		case string:
			fields = strings.Split(data.(string), c.separator)
		case []byte:
			fields = strings.Split(string(data.([]byte)), c.separator)
		case *bytes.Buffer:
			fields = strings.Split(data.(*bytes.Buffer).String(), c.separator)
		default:
			return nil, fmt.Errorf("unsupported data type to split, was: %T", data)
		}
		// drop event
		if subsize > 0 && subsize != len(fields) {
			flow.Log().Debugf("SPLIT: Drop event, subsize/fields=%d/%d", subsize, len(fields))
			continue
		}
		out = append(out, events.NewStringFieldsEvent(evt, fields))
	}
	return out, nil
}
