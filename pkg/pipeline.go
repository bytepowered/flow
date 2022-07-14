package flow

type TagSelector struct {
	Input        string   `toml:"input"`        // 匹配Input的Tag Pattern
	Filters      []string `toml:"filters"`      // 匹配Filter的Tag Pattern
	Transformers []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
	Outputs      []string `toml:"outputs"`      // 匹配Dispatcher的Tag Pattern
}

////

type Definition struct {
	Description string      `toml:"description"` // 路由分组描述
	Selector    TagSelector `toml:"selector"`    // 基于组件Tag的选择器定义
}

////

type Worker func(task func())

type Pipeline struct {
	Input        string
	filters      []Filter
	transformers []Transformer
	outputs      []Output
	worker       Worker
}

func NewPipeline(input string) *Pipeline {
	return &Pipeline{
		Input:        input,
		filters:      make([]Filter, 0, 4),
		transformers: make([]Transformer, 0, 4),
		outputs:      make([]Output, 0, 4),
		worker:       func(t func()) { t() },
	}
}

func (p *Pipeline) AddFilter(f Filter) {
	p.filters = append(p.filters, f)
}

func (p *Pipeline) AddTransformer(t Transformer) {
	p.transformers = append(p.transformers, t)
}

func (p *Pipeline) AddOutput(d Output) {
	p.outputs = append(p.outputs, d)
}

func (p *Pipeline) SetWorker(worker Worker) {
	p.worker = worker
}
