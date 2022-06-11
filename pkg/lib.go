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

var _ fmt.Stringer = new(Header)

// Header 事件Header
type Header struct {
	Time int64  `json:"etimens"` // 用于标识发生事件的时间戳，精确到纳秒
	Tag  string `json:"etag"`    // 用于标识发生事件来源的标签，通常格式为: origin.vendor
	Kind Kind   `json:"ekind"`   // 事件类型，由业务定义
	Id   int64  `json:"eid"`     // 事件ID
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
	// ID 返回事件ID，与 Header.Id 一致。
	ID() int64
	// Header 返回事件Header
	Header() Header
	// Record 返回事件记录对象
	Record() interface{}
}

type Pluginable interface {
	// Tag 返回标识实现对象的标签
	Tag() string
}

// Input 事件输入源
type Input interface {
	Pluginable
	OnRead(ctx context.Context, queue chan<- Event)
}

// Output 事件输出源
type Output interface {
	Pluginable
	OnSend(ctx context.Context, events ...Event)
}

// FilterFunc 执行过滤原始Event的函数；
type FilterFunc func(ctx context.Context, event Event) error

// Filter 原始Event过滤接口
type Filter interface {
	Pluginable
	DoFilter(next FilterFunc) FilterFunc
}

// Transformer 处理Event格式转换
type Transformer interface {
	Pluginable
	DoTransform(ctx context.Context, in []Event) (out []Event, err error)
}
