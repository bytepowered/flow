package flow

var (
	_EventTypeNames = make(map[EventType]string, 8)
)

func SetEventTypeNames(kv map[EventType]string) {
	_EventTypeNames = kv
}

func SetEventTypeName(typ EventType, name string) {
	_EventTypeNames[typ] = name
}

func EventTypeNameOf(typ EventType) string {
	if n, ok := _EventTypeNames[typ]; ok {
		return n
	}
	return "undefined"
}
