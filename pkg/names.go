package flow

var (
	KindNameMapper = make(KindNameValues, 8)
)

func SetKindNameValues(values KindNameValues) {
	KindNameMapper = values
}

type KindNameValues map[Kind]string

func (m KindNameValues) GetName(k Kind) string {
	if v, ok := m[k]; ok {
		return v
	}
	return "undefined"
}

func (m KindNameValues) SetName(k Kind, name string) {
	m[k] = name
}
