package flow

import "context"

// State 表示状态
type State int

func (s State) Is(state State) bool {
	return int(s)&int(state) != 0
}

const (
	RunStateInit     State = 1
	RunStateStartup  State = 2
	RunStateShutdown State = 4
)

type Initable interface {
	// OnInit 初始化组件
	// 此方法执行时，如果返回非nil的error，整个服务启动过程将被终止。
	OnInit() error
}

type Servable interface {
	// Serve 基于Context执行服务；
	// 此方法执行时，如果返回非nil的error，整个服务运行过程将被终止。
	// Serve函数运行时应当处于阻塞状态。如果函数执行并返回，表示服务的停止。
	Serve(ctx context.Context) error
}

type Startup interface {
	// Startup 用于启动组件的生命周期方法；
	// 此方法执行时，如果返回非nil的error，整个服务启动过程将被终止。
	Startup(ctx context.Context) error
}

type Shutdown interface {
	// Shutdown 用于停止组件的生命周期方法；
	// 如果返回非nil的error，将打印日志记录；
	Shutdown(ctx context.Context) error
}

// InputConfig Input的配置化对象
type InputConfig struct {
	QueueSize uint `json:"queue_size"`
}

// Configurable 配置化接口
type Configurable interface {
	// OnConfigure 返回配置对象
	OnConfigure() interface{}
}
