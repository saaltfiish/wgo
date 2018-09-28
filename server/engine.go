package server

import (
	"net"
	"time"
)

type (
	// 转化程序可以把wgo middleware转化为各子package的middleware
	MiddlewareConvFunc func(...interface{}) []interface{}

	EngineFactory func() Engine
	MuxFactory    func() Mux

	// engine
	Engine interface {
		Name() string
		// multiplexer
		Mux() Mux
		SetMux(Mux) Mux
		// start
		Start(net.Listener) error
	}

	// multiplexer
	Mux interface {
		Prepare()
		Use(...interface{}) // middlewares
		SetLogger(interface{})
		Logger() Logger
		SetEngine(Engine)
		Engine() Engine
		Serve(interface{}, interface{})
	}

	// Cookie defines the interface for HTTP cookie.
	Cookie interface {
		// Name returns the name of the cookie.
		Name() string

		// Value returns the value of the cookie.
		Value() string

		// Path returns the path of the cookie.
		Path() string

		// Domain returns the domain of the cookie.
		Domain() string

		// Expires returns the expiry time of the cookie.
		Expires() time.Time

		// Secure indicates if cookie is secured.
		Secure() bool

		// HTTPOnly indicate if cookies is HTTP only.
		HTTPOnly() bool
	}

	// Header defines the interface for HTTP header.
	Header interface {
		// Add adds the key, value pair to the header. It appends to any existing values
		// associated with key.
		Add(string, string)

		// Del deletes the values associated with key.
		Del(string)

		// Set sets the header entries associated with key to the single element value.
		// It replaces any existing values associated with key.
		Set(string, string)

		// Get gets the first value associated with the given key. If there are
		// no values associated with the key, Get returns "".
		Get(string) string

		// Keys returns the header keys.
		Keys() []string

		// Contains checks if the header is set.
		Contains(string) bool
	}

	// URL defines the interface for HTTP request url.
	URL interface {
		// Path returns the request URL path.
		Path() string

		// SetPath sets the request URL path.
		SetPath(string)

		// QueryParam returns the query param for the provided name.
		QueryParam(string) string

		// QueryParam returns the query parameters as map.
		QueryParams() map[string][]string

		// QueryString returns the URL query string.
		QueryString() string
	}
)
