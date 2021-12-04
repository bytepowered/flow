package flow

import (
	"context"
	"time"
)

const (
	TagGlobal = "@global"
)

// EventType 表示Event类型
type EventType uint16

func (e EventType) String() string {
	return EventTypeNameOf(e)
}

// EventState 表示Event状态
type EventState int

func (s EventState) Is(state EventState) bool {
	return int(s)&int(state) != 0
}

// EventHeader 事件Header
type EventHeader struct {
	Time int64     `json:"etimestamp"` // 用于标识发生事件的时间戳，精确到纳秒
	Tag  string    `json:"etag"`       // 用于标识发生事件来源的标签，通常格式为: origin.vendor
	Type EventType `json:"etype"`      // 事件类型，由业务定义
}

// EventRecord 具体Event消息接口
type EventRecord interface {
	// Tag 返回事件标签
	Tag() string
	// Time 返回事件发生时间
	Time() time.Time
	// Header 返回事件Header
	Header() EventHeader
	// Record 返回事件记录对象
	Record() interface{}
	// Frames 返回事件原始数据
	Frames() []byte
}

const (
	EventStateNop   EventState = 0x00000000
	EventStateAsync EventState = 0x00000001
	EventStateSync  EventState = 0x00000002
	//EventStateX     EventState = 0x00000004
	//EventStateXX    EventState = 0x00000008
)

// EventContext 发生Event的上下文
type EventContext interface {
	// Context 返回Context
	Context() context.Context
	// Var 获取Context设定的变量；
	// 属于Context().Value()方法的快捷方式。
	Var(key interface{}) interface{}
	// VarE 获取Context设定的变量，返回变量是否存在。
	// 属于Context().Value()方法的快捷方式。
	VarE(key interface{}) (interface{}, bool)
	//State 返回当前Event的状态
	State() EventState
}

// EventEmitter Event投递接口。当 SourceAdapter 触发事件时，使用 EventEmitter 投递事件。
type EventEmitter interface {
	// Emit 当Adapter接收到Event数据时，调用此方法来投递事件。
	Emit(EventContext, EventRecord)
}

// EventFormatter Event格式处理，用于将字节流转换为事件对象。
type EventFormatter interface {
	Format([]byte) (EventRecord, error)
}

type Component interface {
	// Tag 返回标识实现对象的标签
	Tag() string
	// TypeId 返回实现类型的ID
	TypeId() string
}

// SourceAdapter 数据源适配接口
type SourceAdapter interface {
	Component
	// SetEmitter 适配器触发事件时，调用通过此方法设定的Handler来通知处理事件
	SetEmitter(emitter EventEmitter)
	// Emit 适配器触发事件时，调用通过此方法设定的Handler来通知处理事件
	Emit(ctx EventContext, record EventRecord)
}

// EventFilterFunc 执行过滤原始Event的函数；
type EventFilterFunc func(ctx EventContext, record EventRecord) error

// EventFilter 原始Event过滤接口
type EventFilter interface {
	Component
	DoFilter(next EventFilterFunc) EventFilterFunc
}

// Transformer 处理Event格式转换
type Transformer interface {
	Component
	DoTransform(EventRecord) (EventRecord, error)
}

// Dispatcher Event派发处理接口
type Dispatcher interface {
	Component
	DoDelivery(EventRecord) error
}
