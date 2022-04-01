package flow

type RouterGroupDefinition struct {
	Description string `toml:"description"` // 路由分组描述
	Selector    struct {
		Input        string   `toml:"input"`        // 匹配Input的Tag Pattern
		Filters      []string `toml:"filters"`      // 匹配Filter的Tag Pattern
		Transformers []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
		Outputs      []string `toml:"outputs"`      // 匹配Dispatcher的Tag Pattern
	} `toml:"selector"`
}

type RouterDefinition struct {
	Description  string
	Input        string
	Filters      []string
	Transformers []string
	Outputs      []string
}

type Router struct {
	Input        string
	filters      []Filter
	transformers []Transformer
	outputs      []Output
}

func NewRouter(tag string) *Router {
	return &Router{
		Input:        tag,
		filters:      make([]Filter, 0, 4),
		transformers: make([]Transformer, 0, 4),
		outputs:      make([]Output, 0, 4),
	}
}

func (r *Router) AddFilter(f Filter) {
	r.filters = append(r.filters, f)
}

func (r *Router) AddTransformer(t Transformer) {
	r.transformers = append(r.transformers, t)
}

func (r *Router) AddOutput(d Output) {
	r.outputs = append(r.outputs, d)
}
