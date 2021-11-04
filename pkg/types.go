package flow

var (
	_originNames    = make(map[Origin]string, 8)
	_vendorNames    = make(map[Vendor]string, 8)
	_EventTypeNames = make(map[EventType]string, 8)
)

func RegisterOriginNames(kv map[Origin]string) {
	_originNames = kv
}

func OriginName(origin Origin) string {
	if n, ok := _originNames[origin]; ok {
		return n
	}
	return "undefined"
}

func RegisterVendorNames(kv map[Vendor]string) {
	_vendorNames = kv
}

func VendorName(vendor Vendor) string {
	if n, ok := _vendorNames[vendor]; ok {
		return n
	}
	return "undefined"
}

func RegisterEventTypeNames(kv map[EventType]string) {
	_EventTypeNames = kv
}

func EventTypeName(typ EventType) string {
	if n, ok := _EventTypeNames[typ]; ok {
		return n
	}
	return "undefined"
}
