package rpc

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/transport"
)

type Server struct {
	Mux *ServeMux
}

func (s *Server) Serve(l transport.Listener) error {
	for {
		sess, err := l.Accept()
		if err != nil {
			return err
		}
		go s.Respond(sess)
	}
}

func (s *Server) Respond(sess transport.Session) {
	for {
		ch, err := sess.Accept()
		if err != nil {
			if err == io.EOF {
				return
			}
			panic(err)
		}
		go respond(sess, ch, s.Mux)
	}
}

func respond(sess transport.Session, ch transport.Channel, mux *ServeMux) {
	defer ch.Close()

	framer := &codec.FrameCodec{mux.codec}
	dec := framer.Decoder(ch)

	var call Call
	err := dec.Decode(&call)
	if err != nil {
		log.Println("rpc.Respond:", err)
		return
	}

	call.Decoder = dec
	call.Caller = &Client{
		session: sess,
		codec:   mux.codec,
	}

	header := &ResponseHeader{}
	resp := &responder{
		ch:     ch,
		c:      framer,
		header: header,
	}

	handler := mux.Handler(call.Selector)
	if handler == nil {
		resp.Return(fmt.Errorf("handler does not exist for this selector: %s", call.Selector))
		return
	}

	handler.RespondRPC(resp, &call)
}

type ServeMux struct {
	handlers map[string]Handler
	codec    codec.Codec
	mu       sync.Mutex
}

var DefaultServeMux = &ServeMux{
	handlers: make(map[string]Handler),
	codec:    codec.JSONCodec{},
}

func NewServeMux(codec codec.Codec) *ServeMux {
	return &ServeMux{
		handlers: make(map[string]Handler),
		codec:    codec,
	}
}

// TODO: Revisit
// Bind makes a Handler accessible at a selector. Non-Handlers
// are exported with MustExport.
func (m *ServeMux) Bind(selector string, v interface{}) {
	var handlers map[string]Handler
	if h, ok := v.(Handler); ok {
		handlers = map[string]Handler{"": h}
	} else {
		handlers = MustExport(v)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for p, h := range handlers {
		if selector != "" && p != "" {
			p = strings.Join([]string{selector, p}, ".")
		} else {
			p = strings.Join([]string{selector, p}, "")
		}
		m.handlers[p] = h
	}
}

func Bind(selector string, v interface{}) {
	DefaultServeMux.Bind(selector, v)
}

func (m *ServeMux) Handler(selector string) Handler {
	var handler Handler
	m.mu.Lock()
	for k, v := range m.handlers {
		if (strings.HasSuffix(k, "/") && strings.HasPrefix(selector, k)) || selector == k {
			handler = v
			break
		}
	}
	m.mu.Unlock()
	return handler
}

func (m *ServeMux) RespondRPC(Responder, *Call) {
	// TODO
}

// func (m *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	websocket.Handler(func(ws *websocket.Conn) {
// 		ws.PayloadType = websocket.BinaryFrame
// 		sess := mux.NewSession(r.Context(), ws)
// 		for {
// 			ch, err := sess.Accept()
// 			if err != nil {
// 				if err == io.EOF {
// 					return
// 				}
// 				panic(err)
// 			}
// 			go Respond(sess, ch, m)
// 		}
// 	}).ServeHTTP(w, r)
// }
