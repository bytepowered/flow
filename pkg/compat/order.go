package compat

import (
	"github.com/bytepowered/runv"
)

var (
	_ runv.Liveorder = OrderedPlugin{}
	_ runv.Liveorder = OrderedSource{}
)

const (
	BaseOrderPlugin = 1000
	BaseOrderSource = 1000
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

// OrderedSource Source生命周期顺序，启动排后，关闭优先
type OrderedSource struct{}

func (OrderedSource) Order(state runv.State) int {
	switch state {
	case runv.StateShutdown:
		return -1 * BaseOrderSource
	case runv.StateStartup:
		return BaseOrderSource
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
