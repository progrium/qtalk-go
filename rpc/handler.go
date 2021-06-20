package rpc

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Handler interface {
	RespondRPC(Responder, *Call)
}

type HandlerFunc func(Responder, *Call)

func (f HandlerFunc) RespondRPC(resp Responder, call *Call) {
	f(resp, call)
}

// NotFoundHandler returns a simple handler that returns an error "not found"
func NotFoundHandler() Handler {
	return HandlerFunc(func(r Responder, c *Call) {
		r.Return(fmt.Errorf("not found: %s", c.Selector))
	})
}

type RespondMux struct {
	m  map[string]muxEntry
	es []muxEntry // slice of entries sorted from longest to shortest.
	mu sync.RWMutex
}

type muxEntry struct {
	h       Handler
	pattern string
}

// cleanSelector returns the canonical selector for s, normalizing . separators to /.
func cleanSelector(s string) string {
	if s == "" {
		return "/"
	}
	if s[0] != '/' {
		s = "/" + s
	}
	s = strings.ReplaceAll(s, ".", "/")
	return s
}

func NewRespondMux() *RespondMux { return new(RespondMux) }

func (m *RespondMux) RespondRPC(r Responder, c *Call) {
	h, _ := m.Handler(c)
	h.RespondRPC(r, c)
}

func (m *RespondMux) Handler(c *Call) (h Handler, pattern string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	h, pattern = m.Match(c.Selector)
	if h == nil {
		h, pattern = NotFoundHandler(), ""
	}
	return
}

func (m *RespondMux) Remove(selector string) (h Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	selector = cleanSelector(selector)
	h = m.m[selector].h
	delete(m.m, selector)

	return
}

// Find a handler on a handler map given a selector string.
// Most-specific (longest) pattern wins.
func (m *RespondMux) Match(selector string) (h Handler, pattern string) {
	selector = cleanSelector(selector)

	// Check for exact match first.
	v, ok := m.m[selector]
	if ok {
		return v.h, v.pattern
	}

	// Check for longest valid match.  m.es contains all patterns
	// that end in / sorted from longest to shortest.
	for _, e := range m.es {
		if strings.HasPrefix(selector, e.pattern) {
			return e.h, e.pattern
		}
	}
	return nil, ""
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (m *RespondMux) Handle(pattern string, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pattern == "" {
		panic("rpc: invalid pattern")
	}
	pattern = cleanSelector(pattern)

	if handler == nil {
		panic("rpc: nil handler")
	}
	if _, exist := m.m[pattern]; exist {
		panic("rpc: multiple registrations for " + pattern)
	}

	if m.m == nil {
		m.m = make(map[string]muxEntry)
	}
	e := muxEntry{h: handler, pattern: pattern}
	m.m[pattern] = e
	if pattern[len(pattern)-1] == '/' {
		m.es = appendSorted(m.es, e)
	}
}

func appendSorted(es []muxEntry, e muxEntry) []muxEntry {
	n := len(es)
	i := sort.Search(n, func(i int) bool {
		return len(es[i].pattern) < len(e.pattern)
	})
	if i == n {
		return append(es, e)
	}
	// we now know that i points at where we want to insert
	es = append(es, muxEntry{}) // try to grow the slice in place, any entry works.
	copy(es[i+1:], es[i:])      // Move shorter entries down
	es[i] = e
	return es
}
