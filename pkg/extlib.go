package flow

// InputConfig Input的配置化对象
type InputConfig struct {
	QueueSize uint `json:"queue_size"`
}

// Configurable 配置化接口
type Configurable interface {
	// OnConfigure 返回配置对象
	OnConfigure() interface{}
}
