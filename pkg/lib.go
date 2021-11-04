package flow

import (
	"context"
	"reflect"
)

// Vendor 表示事件来源厂商
type Vendor uint16

// EventType 表示事件类型
type EventType uint16

// EventHeader 行情数据Header
type EventHeader struct {
	RecvTime  int64     `json:"recvTime"`  // 接收数据的系统时间戳，精确到纳秒
	Origin    uint8     `json:"origin"`    // 来源类型
	Vendor    Vendor    `json:"vendor"`    // 所属厂商
	EventType EventType `json:"eventType"` // 事件类型
}

// Event 事件消息
type Event interface {
	Header() EventHeader
	Object() interface{}
	Type() reflect.Type
}

// EventContext 发生事件的上下文
type EventContext interface {
	// Context 返回Context
	Context() context.Context

	// GetVar 获取Context设定的变量；
	// 属于Context().Value()方法的快捷方式。
	GetVar(key interface{}) interface{}

	// GetVarE 获取Context设定的变量，返回变量是否存在。
	// 属于Context().Value()方法的快捷方式。
	GetVarE(key interface{}) (interface{}, bool)

	//Async 返回当前事件处理的调用过程是否为异步
	Async() bool
}

// EventHandler 事件处理接口
type EventHandler interface {
	// OnReceived 当Adapter接收到数据时，调用此方法来解析消息的Header
	OnReceived(ctx EventContext, header EventHeader, packet []byte)
}

// EventFilterInvoker 执行过滤原始数据操作的函数
type EventFilterInvoker func(header EventHeader) error

// EventFilter 行情原始数据过滤接口
type EventFilter interface {
	DoFilter(next EventFilterInvoker) EventFilterInvoker
}

// TypedEventFilter 针对特定类型原始数据进行过滤接口
type TypedEventFilter interface {
	// EffectON 在特定事件类型下生效
	EffectON() EventType
	EventFilter
}

// Transformer 处理事件格式转换
type Transformer interface {
	// ActiveON 在特定类型下生效
	ActiveON() Vendor
	DoTransform(header EventHeader, packet []byte) (Event, error)
}

// Adapter 数据源适配接口
type Adapter interface {
	AdapterId() string
	SetEventHandler(handler EventHandler)
}

// Dispatcher 事件派发处理接口
type Dispatcher interface {
	DispatcherId() string
	Dispatch(message Event) error
}
