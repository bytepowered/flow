package filter

import flow "github.com/bytepowered/flow/v2/pkg"

var _ flow.Filter = new(GlobalFilter)

type GlobalFilter struct {
	TypeTag string
}

func (f *GlobalFilter) Tag() string {
	return flow.TagGLOBAL + f.TypeTag
}

func (f *GlobalFilter) TypeId() string {
	return f.TypeTag
}

func (f *GlobalFilter) DoFilter(next flow.FilterFunc) flow.FilterFunc {
	panic("child filter not implement yet")
}
