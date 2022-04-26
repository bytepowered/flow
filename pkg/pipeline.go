package flow

type PipelineSelector struct {
	InputExpr        string   `toml:"input"`        // 匹配Input的Tag Pattern
	FiltersExpr      []string `toml:"filters"`      // 匹配Filter的Tag Pattern
	TransformersExpr []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
	OutputsExpr      []string `toml:"outputs"`      // 匹配Dispatcher的Tag Pattern
}

type PipelineDefinition struct {
	Description string           `toml:"description"` // 路由分组描述
	Selector    PipelineSelector `toml:"selector"`
}

type PipelineDescriptor struct {
	Description  string
	Input        string
	Filters      []string
	Transformers []string
	Outputs      []string
}

type Pipeline struct {
	Input        string
	filters      []Filter
	transformers []Transformer
	outputs      []Output
}

func NewPipeline(input string) *Pipeline {
	return &Pipeline{
		Input:        input,
		filters:      make([]Filter, 0, 4),
		transformers: make([]Transformer, 0, 4),
		outputs:      make([]Output, 0, 4),
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
