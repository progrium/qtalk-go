package rpc

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// A Handler responds to an RPC request.
//
// RespondRPC should use Call to receive at least one input argument value, then use
// Responder to return a value or continue. Since an input argument value is always
// sent to the handler, a call to Receive on the Call value shoud always be done otherwise
// the call will block. You can call Receive with nil to discard the input value. If
// Responder is not used, a default value of nil is returned.
type Handler interface {
	RespondRPC(Responder, *Call)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as RPC handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(Responder, *Call)

// RespondRPC calls f(resp, call).
func (f HandlerFunc) RespondRPC(resp Responder, call *Call) {
	f(resp, call)
}

// NotFoundHandler returns a simple handler that returns an error "not found".
func NotFoundHandler() Handler {
	return HandlerFunc(func(r Responder, c *Call) {
		r.Return(fmt.Errorf("not found: %s", c.Selector))
	})
}

// RespondMux is an RPC call multiplexer. It matches the selector of each incoming call against a list of
// registered selector patterns and calls the handler for the pattern that most closely matches the selector.
//
// RespondMux also takes care of normalizing the selector to a path form "/foo/bar", allowing you to use
// this or the more conventional RPC dot form "foo.bar".
//
// Patterns match exact incoming selectors, or can end with a "/" or "." to indicate handling any selectors
// beginning with this pattern. Longer patterns take precedence over shorter ones, so that if there are
// handlers registered for both "foo." and "foo.bar.", the latter handler will be called for selectors
// beginning "foo.bar." and the former will receive calls for any other selectors prefixed with "foo.".
//
// Since RespondMux is also a Handler, you can use them for submuxing. If a pattern matches a handler that
// is a RespondMux, it will trim the matching selector prefix before matching against the sub RespondMux.
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

// NewRespondMux allocates and returns a new RespondMux.
func NewRespondMux() *RespondMux { return new(RespondMux) }

// RespondRPC dispatches the call to the handler whose pattern most closely matches the selector.
func (m *RespondMux) RespondRPC(r Responder, c *Call) {
	h, _ := m.Handler(c)
	h.RespondRPC(r, c)
}

// Handler returns the handler to use for the given call, consulting
// c.Selector. It always returns a non-nil handler.
//
// If there is no registered handler that applies to the request, Handler
// returns a "not found" handler and an empty pattern.
func (m *RespondMux) Handler(c *Call) (h Handler, pattern string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	h, pattern = m.Match(c.Selector)
	if h == nil {
		h, pattern = NotFoundHandler(), ""
	}
	return
}

// Remove removes and returns the handler for the selector.
func (m *RespondMux) Remove(selector string) (h Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	selector = cleanSelector(selector)
	h = m.m[selector].h
	delete(m.m, selector)

	return
}

// Match finds a handler given a selector string.
// Most-specific (longest) pattern wins. If a pattern handler
// is a submux, it will call Match with the selector minus the
// pattern.
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

	// Check for any prefix match that has a handler
	// that is also a matcher (ie is a submuxer)
	for _, e := range m.m {
		if strings.HasPrefix(selector, e.pattern) {
			m, ok := e.h.(interface {
				Match(selector string) (h Handler, pattern string)
			})
			if ok {
				return m.Match(strings.TrimPrefix(selector, e.pattern))
			}
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
