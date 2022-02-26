package flow

var (
	_KindNameRegistry = make(map[Kind]string, 8)
)

func GetKindNames() map[Kind]string {
	return _KindNameRegistry
}

func SetAllKindNames(kv map[Kind]string) {
	_KindNameRegistry = kv
}

func SetKindName(typ Kind, name string) {
	_KindNameRegistry[typ] = name
}

func GetKindNameOf(typ Kind) string {
	if n, ok := _KindNameRegistry[typ]; ok {
		return n
	}
	return "undefined"
}
