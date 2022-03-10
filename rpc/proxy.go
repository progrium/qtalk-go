package rpc

import "io"

// ProxyHandler returns a handler that tries its best to proxy the
// call to the dst Client, regardless of call style and assuming the
// same encoding.
func ProxyHandler(dst *Client) Handler {
	return HandlerFunc(func(r Responder, c *Call) {
		ch, err := dst.Session.Open(c.Context)
		if err != nil {
			r.Return(err)
			return
		}

		framer := &FrameCodec{Codec: dst.codec}
		enc := framer.Encoder(ch)
		err = enc.Encode(CallHeader{
			Selector: c.Selector,
		})
		if err != nil {
			ch.Close()
			r.Return(err)
			return
		}

		go func() {
			io.Copy(ch, c.ch)
			ch.CloseWrite()
		}()
		go func() {
			io.Copy(c.ch, ch)
			c.ch.Close()
		}()

		r.(*responder).responded = true
		r.(*responder).header.Continue = true
	})
}
