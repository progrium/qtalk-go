package rpc

type Handler interface {
	RespondRPC(Responder, *Call)
}

type HandlerFunc func(Responder, *Call)

func (f HandlerFunc) RespondRPC(resp Responder, call *Call) {
	f(resp, call)
}
