# qtalk-go

[![GoDoc](https://godoc.org/github.com/progrium/qtalk-go?status.svg)](https://godoc.org/github.com/progrium/qtalk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/progrium/qtalk-go)](https://goreportcard.com/report/github.com/progrium/qtalk-go)
<a href="https://twitter.com/progriumHQ" title="@progriumHQ on Twitter"><img src="https://img.shields.io/badge/twitter-@progriumHQ-55acee.svg" alt="@progriumHQ on Twitter"></a>
<a href="https://github.com/progrium/qtalk-go/discussions" title="Project Forum"><img src="https://img.shields.io/badge/community-forum-ff69b4.svg" alt="Project Forum"></a>
<a href="https://github.com/sponsors/progrium" title="Sponsor Project"><img src="https://img.shields.io/static/v1?label=sponsor&message=%E2%9D%A4&logo=GitHub" alt="Sponsor Project" /></a>
------
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
