package flow

var (
	_ Adapter = new(DeliverAdapter)
)

type DeliverAdapter struct {
	id      string
	deliver EventDeliverFunc
}

func NewDeliverAdapter(id string) *DeliverAdapter {
	return &DeliverAdapter{id: id}
}

func (a *DeliverAdapter) AdapterId() string {
	return a.id
}

func (a *DeliverAdapter) SetEventDeliverFunc(df EventDeliverFunc) {
	a.deliver = df
}

func (a *DeliverAdapter) Deliver(ctx EventContext, header EventHeader, packet []byte) {
	a.deliver.Deliver(ctx, header, packet)
}
