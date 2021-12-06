package flow

func LookupStateOf(root string, typeid string) (reason string, disabled bool) {
	state := struct {
		Disabled bool `toml:"disabled"`
	}{}
	ok, err := RootConfigOf(root).Lookup(typeid, &state)
	if ok && err != nil {
		return "config-error:" + err.Error(), true
	}
	if !ok {
		return "config-notfound", true
	}
	if state.Disabled {
		return "set-disabled", true
	}
	return "active", false
}
