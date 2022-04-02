package flow

type PipelineDefinition struct {
	Description string `toml:"description"` // 路由分组描述
	Selector    struct {
		Input        string   `toml:"input"`        // 匹配Input的Tag Pattern
		Filter       []string `toml:"filters"`      // 匹配Filter的Tag Pattern
		Transformers []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
		Outputs      []string `toml:"outputs"`      // 匹配Dispatcher的Tag Pattern
	} `toml:"selector"`
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

func (r *Pipeline) AddFilter(f Filter) {
	r.filters = append(r.filters, f)
}

func (r *Pipeline) AddTransformer(t Transformer) {
	r.transformers = append(r.transformers, t)
}

func (r *Pipeline) AddOutput(d Output) {
	r.outputs = append(r.outputs, d)
}
