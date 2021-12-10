package flow

import (
	"context"
	"time"
)

const (
	TagGLOBAL = "@globals."
)

// Kind 表示Event类型
type Kind uint16

func (e Kind) String() string {
	return KindNameOf(e)
}

// State 表示Event状态
type State int

func (s State) Is(state State) bool {
	return int(s)&int(state) != 0
}

// Header 事件Header
type Header struct {
	Time int64  `json:"etimens"` // 用于标识发生事件的时间戳，精确到纳秒
	Tag  string `json:"etag"`       // 用于标识发生事件来源的标签，通常格式为: origin.vendor
	Kind Kind   `json:"ekind"`      // 事件类型，由业务定义
}

// Event 具体Event消息接口
type Event interface {
	// Tag 返回事件标签
	Tag() string
	// Time 返回事件发生时间
	Time() time.Time
	// Header 返回事件Header
	Header() Header
	// Record 返回事件记录对象
	Record() interface{}
	// Frames 返回事件原始数据
	Frames() []byte
}

const (
	StateNop   State = 0x00000000
	StateAsync State = 0x00000001
	StateSync  State = 0x00000002
	//StateX     State = 0x00000004
	//StateXX    State = 0x00000008
)

// StateContext 发生Event的上下文
type StateContext interface {
	// Context 返回Context
	Context() context.Context
	// Var 获取Context设定的变量；
	// 属于Context().Value()方法的快捷方式。
	Var(key interface{}) interface{}
	// VarE 获取Context设定的变量，返回变量是否存在。
	// 属于Context().Value()方法的快捷方式。
	VarE(key interface{}) (interface{}, bool)
	//State 返回当前Event的状态
	State() State
}

// Emitter Event投递接口。当 Source 触发事件时，使用 Emitter 投递事件。
type Emitter interface {
	// Emit 当Adapter接收到Event数据时，调用此方法来投递事件。
	Emit(StateContext, Event)
}

type Component interface {
	// Tag 返回标识实现对象的标签
	Tag() string
	// TypeId 返回实现类型的ID
	TypeId() string
}

// Formatter Event格式处理，用于将字节流转换为事件对象。
type Formatter interface {
	OnInit(args interface{}) error
	DoFormat(ctx context.Context, srctag string, data []byte) (Event, error)
}

// Source 数据源适配接口
type Source interface {
	Component
	// SetEmitter 适配器触发事件时，调用通过此方法设定的Handler来通知处理事件
	SetEmitter(emitter Emitter)
	// Emit 适配器触发事件时，调用通过此方法设定的Handler来通知处理事件
	Emit(ctx StateContext, event Event)
}

// FilterFunc 执行过滤原始Event的函数；
type FilterFunc func(ctx StateContext, event Event) error

// Filter 原始Event过滤接口
type Filter interface {
	Component
	DoFilter(next FilterFunc) FilterFunc
}

// Transformer 处理Event格式转换
type Transformer interface {
	Component
	DoTransform(Event) (Event, error)
}

// Output Event派发处理接口
type Output interface {
	Component
	Send(Event) error
}
