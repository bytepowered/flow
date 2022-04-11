package flow

import (
	"context"
	"fmt"
	"time"
)

// Kind 表示Event类型
type Kind uint16

func (k Kind) String() string {
	return KindNameMapper.GetName(k)
}

// State 表示Event状态
type State int

func (s State) Is(state State) bool {
	return int(s)&int(state) != 0
}

var _ fmt.Stringer = new(Header)

// Header 事件Header
type Header struct {
	Time int64  `json:"etimens"` // 用于标识发生事件的时间戳，精确到纳秒
	Tag  string `json:"etag"`    // 用于标识发生事件来源的标签，通常格式为: origin.vendor
	Kind Kind   `json:"ekind"`   // 事件类型，由业务定义
}

func (h Header) String() string {
	return fmt.Sprintf(`time: %s, tag: %s, kind: %s`,
		time.UnixMilli(time.Duration(h.Time).Milliseconds()), h.Tag, h.Kind)
}

// Event 具体Event消息接口
type Event interface {
	// Tag 返回事件标签。与 Header.Tag 一致。
	Tag() string
	// Kind 返回事件类型。与 Header.Kind 一致。
	Kind() Kind
	// Time 返回事件发生时间。与 Header.Time 一致。
	Time() time.Time
	// Header 返回事件Header
	Header() Header
	// Record 返回事件记录对象
	Record() interface{}
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

type Plugin interface {
	// Tag 返回标识实现对象的标签
	Tag() string
}

// Input 事件输入源
type Input interface {
	Plugin
	OnReceived(ctx context.Context, queue chan<- Event)
}

// Output 事件输出源
type Output interface {
	Plugin
	OnSend(ctx context.Context, events ...Event)
}

// FilterFunc 执行过滤原始Event的函数；
type FilterFunc func(ctx StateContext, event Event) error

// Filter 原始Event过滤接口
type Filter interface {
	Plugin
	DoFilter(next FilterFunc) FilterFunc
}

// Formatter Event格式处理，用于将字节流转换为事件对象。
type Formatter interface {
	DoFormat(ctx context.Context, intag string, data []byte) (Event, error)
}

// Transformer 处理Event格式转换
type Transformer interface {
	Plugin
	DoTransform(ctx StateContext, in []Event) (out []Event, err error)
}
