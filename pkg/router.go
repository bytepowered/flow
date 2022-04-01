package flow

type RouteGroupDefinition struct {
	Description string `toml:"description"` // 路由分组描述
	Selector    struct {
		Input           string   `toml:"input"`        // 匹配Input的Tag Pattern
		FilterTags      []string `toml:"filters"`      // 匹配Filter的Tag Pattern
		TransformerTags []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
		OutputTags      []string `toml:"outputs"`      // 匹配Dispatcher的Tag Pattern
	} `toml:"selector"`
}

type RouteMapper struct {
	Description  string
	Input        string
	Filters      []string
	Transformers []string
	Outputs      []string
}

type RouteDescriptor struct {
	Input        string
	filters      []Filter
	transformers []Transformer
	outputs      []Output
}

func NewRouter(tag string) *RouteDescriptor {
	return &RouteDescriptor{
		Input:        tag,
		filters:      make([]Filter, 0, 4),
		transformers: make([]Transformer, 0, 4),
		outputs:      make([]Output, 0, 4),
	}
}

func (r *RouteDescriptor) AddFilter(f Filter) {
	r.filters = append(r.filters, f)
}

func (r *RouteDescriptor) AddTransformer(t Transformer) {
	r.transformers = append(r.transformers, t)
}

func (r *RouteDescriptor) AddOutput(d Output) {
	r.outputs = append(r.outputs, d)
}
