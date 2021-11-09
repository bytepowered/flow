package flow

var (
	_ Adapter = new(AbcAdapter)
)

type AbcAdapter struct {
	id      string
	deliver EventDeliverFunc
}

func NewAbcAdapter(id string) *AbcAdapter {
	return &AbcAdapter{id: id}
}

func (a *AbcAdapter) AdapterId() string {
	return a.id
}

func (a *AbcAdapter) SetEventDeliverFunc(df EventDeliverFunc) {
	a.deliver = df
}

func (a *AbcAdapter) Deliver(ctx EventContext, header EventHeader, packet []byte) {
	a.deliver.Deliver(ctx, header, packet)
}
