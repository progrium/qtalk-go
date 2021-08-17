# qtalk-go

qtalk-go is a versatile RPC and IO stream based IPC stack for Go: 

 * client *or* server can make RPC calls to the other end
 * calls can be unary or streaming for multiple inputs/outputs
 * pluggable data codecs for flexible object stream marshaling
 * RPC calls designed to optionally become full-duplex byte streams
 * muxing layer based on subset of SSH (qmux) and soon optionally QUIC
 * qmux allows any `io.ReadWriteCloser` transport, including STDIO
 * API inspired by `net/http` with easy function/method export on top
 * supports passing remote callbacks over RPC

The goal was to come up with the most minimal design for the most flexibility
in how you want to communicate between processes. 

## Getting Started 
```
$ go get github.com/progrium/qtalk-go
```

## License

MIT