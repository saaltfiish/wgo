package standard

import "net/http"

type (
	// Header implements `server.Header`.
	Header struct {
		http.Header
	}
)

// Add implements `server.Header#Add` function.
func (h *Header) Add(key, val string) {
	h.Header.Add(key, val)
}

// Del implements `server.Header#Del` function.
func (h *Header) Del(key string) {
	h.Header.Del(key)
}

// Set implements `server.Header#Set` function.
func (h *Header) Set(key, val string) {
	h.Header.Set(key, val)
}

// Get implements `server.Header#Get` function.
func (h *Header) Get(key string) string {
	return h.Header.Get(key)
}

// Keys implements `server.Header#Keys` function.
func (h *Header) Keys() (keys []string) {
	keys = make([]string, len(h.Header))
	i := 0
	for k := range h.Header {
		keys[i] = k
		i++
	}
	return
}

// Contains implements `server.Header#Contains` function.
func (h *Header) Contains(key string) bool {
	_, ok := h.Header[key]
	return ok
}

func (h *Header) reset(hdr http.Header) {
	h.Header = hdr
}
