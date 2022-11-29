package mux

import (
	"context"
	"io"
	"sync"
)

// Proxy accepts channels on src then opens a channel on dst and performs
// an io.Copy in both directions in goroutines. Proxy returns non-EOF errors
// from src.Accept, nil on EOF, and any errors from dst.Open after closing
// the accepted channel from src.
func Proxy(dst, src Session) error {
	for {
		ctx := context.Background()
		a, err := src.Accept()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		b, err := dst.Open(ctx)
		if err != nil {
			a.Close()
			return err
		}
		go proxy(a, b)
	}
}

func proxy(a, b Channel) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		io.Copy(a, b)
		a.CloseWrite()
		wg.Done()
	}()
	go func() {
		io.Copy(b, a)
		b.CloseWrite()
		wg.Done()
	}()
	wg.Wait()
	a.Close()
	b.Close()
}
