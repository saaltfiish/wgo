// Package wrpc provides ...
package wrpc

import (
	"strings"

	"google.golang.org/grpc/metadata"
)

type Header struct {
	metadata.MD
}

// Add implements `server.Header#Add` function.
func (h *Header) Add(key, val string) {
	key = strings.ToLower(key)
	h.MD[key] = append(h.MD[key], val)
}

// Del implements `server.Header#Del` function.
func (h *Header) Del(key string) {
	key = strings.ToLower(key)
	delete(h.MD, key)
}

// Set implements `server.Header#Set` function.
func (h *Header) Set(key, val string) {
	key = strings.ToLower(key)
	h.Del(key)
	h.Add(key, val)
}

// Get implements `server.Header#Get` function.
func (h *Header) Get(key string) string {
	if h == nil {
		return ""
	}
	v := h.MD[strings.ToLower(key)]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// Keys implements `server.Header#Keys` function.
func (h *Header) Keys() (keys []string) {
	keys = make([]string, len(h.MD))
	i := 0
	for k := range h.MD {
		keys[i] = k
		i++
	}
	return
}

// Contains implements `server.Header#Contains` function.
func (h *Header) Contains(key string) bool {
	_, ok := h.MD[strings.ToLower(key)]
	return ok
}

func (h *Header) reset(hdr metadata.MD) {
	h.MD = hdr
}
