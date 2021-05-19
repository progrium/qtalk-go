package rpc

import "github.com/progrium/qtalk-go/transport"

type ResponseHeader struct {
	Error    *string
	Continue bool // after parsing response, keep stream open for whatever protocol
}

type Response struct {
	ResponseHeader

	Reply   interface{}
	Channel transport.Channel
}
