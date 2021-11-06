package flow

import (
	"context"
	"reflect"
)

// Origin 表示Event来源
type Origin uint8

func (o Origin) String() string {
	return OriginName(o)
}

// Vendor 表示Event来源厂商
type Vendor uint16

func (v Vendor) String() string {
	return VendorName(v)
}

// EventType 表示Event类型
type EventType uint16

func (e EventType) String() string {
	return EventTypeName(e)
}

// EventHeader 行情数据Header
type EventHeader struct {
	RecvTime  int64     `json:"recvTime"`  // 接收数据的系统时间戳，精确到纳秒
	Origin    Origin    `json:"origin"`    // 来源类型
	Vendor    Vendor    `json:"vendor"`    // 所属厂商
	EventType EventType `json:"eventType"` // Event类型
}

// Event 具体Event消息接口
type Event interface {
	Header() EventHeader
	Object() interface{}
	Type() reflect.Type
}

// EventContext 发生Event的上下文
type EventContext interface {
	// Context 返回Context
	Context() context.Context

	// GetVar 获取Context设定的变量；
	// 属于Context().Value()方法的快捷方式。
	GetVar(key interface{}) interface{}

	// GetVarE 获取Context设定的变量，返回变量是否存在。
	// 属于Context().Value()方法的快捷方式。
	GetVarE(key interface{}) (interface{}, bool)

	//Async 返回当前Event处理的调用过程是否为异步
	Async() bool
}

// EventHandler Event处理接口
type EventHandler interface {
	// OnReceived 当Adapter接收到Event数据时，调用此方法来处理。
	//    ctx Event上下文
	//    header EventHeader
	//    packet Event负载部分的数据。应当不包含Header数据。
	OnReceived(ctx EventContext, header EventHeader, packet []byte)
}

// EventFilterInvoker 执行过滤原始Event的函数；
type EventFilterInvoker func(header EventHeader) error

// EventFilter 原始Event过滤接口
type EventFilter interface {
	DoFilter(next EventFilterInvoker) EventFilterInvoker
}

// TypedEventFilter 针对特定类型原始数据进行过滤接口
type TypedEventFilter interface {
	// EffectON 在特定Event类型下生效
	EffectON() EventType
	// DoFilter 执行Event过滤
	DoFilter(next EventFilterInvoker) EventFilterInvoker
}

// Transformer 处理Event格式转换
type Transformer interface {
	// ActiveON 在特定类型下生效
	ActiveON() Vendor
	// DoTransform 执行Event格式转换
	DoTransform(header EventHeader, packet []byte) (Event, error)
}

// Adapter 数据源适配接口
type Adapter interface {
	// AdapterId 适配接口实现具体类型的ID
	AdapterId() string
	// SetEventHandler 适配器触发事件时，调用通过此方法设定的Handler来通知处理事件
	SetEventHandler(handler EventHandler)
}

// Dispatcher Event派发处理接口
type Dispatcher interface {
	// DispatcherId 派发处理接口实现具体类型的ID
	DispatcherId() string
	// Dispatch 执行Event派发处理
	Dispatch(message Event) error
}
