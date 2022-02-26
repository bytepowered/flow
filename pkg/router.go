package flow

type GroupRouter struct {
	Description string `toml:"description"` // 路由分组描述
	Selector    struct {
		InputTags       []string `toml:"inputs"`       // 匹配Input的Tag Pattern
		FilterTags      []string `toml:"filters"`      // 匹配Filter的Tag Pattern
		TransformerTags []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
		OutputTags      []string `toml:"outputs"`      // 匹配Dispatcher的Tag Pattern
	} `toml:"selector"`
}

type TaggedRouter struct {
	description     string
	InputTag        string
	FilterTags      []string
	TransformerTags []string
	OutputTags      []string
}

type Router struct {
	filters      []Filter
	transformers []Transformer
	outputs      []Output
}

func NewRouter() *Router {
	return &Router{
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
