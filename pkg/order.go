package flow

import (
	"github.com/bytepowered/runv"
)

var (
	_ runv.Liveorder = OrderedPlugin{}
	_ runv.Liveorder = OrderedInput{}
	_ runv.Liveorder = OrderedOutput{}
)

const (
	BaseOrderPlugin = 1000
	BaseOrderInput  = 1000
	BaseOrderOutput = 100
)

// OrderedPlugin 插件生命周期顺序，启动优先，关闭排后
type OrderedPlugin struct{}

func (OrderedPlugin) Order(state runv.State) int {
	switch state {
	case runv.StateShutdown:
		return BaseOrderPlugin
	case runv.StateStartup:
		return -1 * BaseOrderPlugin
	default:
		return 0
	}
}

// OrderedInput Input生命周期顺序，启动排后，关闭优先
type OrderedInput struct{}

func (OrderedInput) Order(state runv.State) int {
	switch state {
	case runv.StateShutdown:
		return -1 * BaseOrderInput
	case runv.StateStartup:
		return BaseOrderInput
	default:
		return 0
	}
}

// OrderedOutput Output生命周期顺序，启动排前，关闭靠后
type OrderedOutput struct{}

func (OrderedOutput) Order(state runv.State) int {
	switch state {
	case runv.StateShutdown:
		return BaseOrderOutput
	case runv.StateStartup:
		return -1 * BaseOrderOutput
	default:
		return 0
	}
}
