package flow

var (
	_KindNames = make(map[Kind]string, 8)
)

func SetKindNames(kv map[Kind]string) {
	_KindNames = kv
}

func SetKindName(typ Kind, name string) {
	_KindNames[typ] = name
}

func KindNameOf(typ Kind) string {
	if n, ok := _KindNames[typ]; ok {
		return n
	}
	return "undefined"
}
