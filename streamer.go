package arf

type InStreamer[T any] interface {
	Recv() (T, error)
}

type OutStreamer[T any] interface {
	Send(T) error
	Close() error
}

type InOutStreamer[I, O any] interface {
	InStreamer[I]
	OutStreamer[O]
}

func MakeInOutStream[I, O any](c Context) InOutStreamer[I, O] {
	return &inOutStream[I, O]{
		inStream:  MakeInStream[I](c).(*inStream[I]),
		outStream: MakeOutStream[O](c).(*outStream[O]),
	}
}

func MakeInStream[I any](c Context) InStreamer[I] {
	return &inStream[I]{c: c}
}

func MakeOutStream[O any](c Context) OutStreamer[O] {
	return &outStream[O]{c: c}
}

type inOutStream[I, O any] struct {
	*inStream[I]
	*outStream[O]
}

type inStream[I any] struct {
	c Context
}

func (i inStream[I]) Recv() (res I, err error) {
	var val any
	val, err = i.c.Recv()
	if err != nil {
		return
	}
	return val.(I), nil
}

type outStream[O any] struct {
	c Context
}

func (o outStream[O]) Send(t O) error {
	return o.c.Send(t)
}

func (o outStream[O]) Close() error {
	return o.c.EndSend()
}
