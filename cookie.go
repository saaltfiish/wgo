package wgo

import "time"

type (
	// Cookie defines the HTTP cookie.
	cookie struct {
		name     string
		value    string
		path     string
		domain   string
		expires  time.Time
		secure   bool
		httpOnly bool
		err      error
	}
)

func NewCookie(name, value, path, domain string, expires time.Time, secure, httpOnly bool) *cookie {
	return &cookie{
		name:     name,
		value:    value,
		path:     path,
		domain:   domain,
		expires:  expires,
		secure:   secure,
		httpOnly: httpOnly,
	}
}

// Name returns the cookie name.
func (c *cookie) Name() string {
	return c.name
}

// SetName sets cookie name.
func (c *cookie) SetName(name string) {
	c.name = name
}

// Value returns the cookie value.
func (c *cookie) Value() string {
	return c.value
}

// SetValue sets the cookie value.
func (c *cookie) SetValue(value string) *cookie {
	c.value = value
	return c
}

// Path returns the cookie path.
func (c *cookie) Path() string {
	return c.path
}

// SetPath sets the cookie path.
func (c *cookie) SetPath(path string) *cookie {
	c.path = path
	return c
}

// Domain returns the cookie domain.
func (c *cookie) Domain() string {
	return c.domain
}

// SetDomain sets the cookie domain.
func (c *cookie) SetDomain(domain string) *cookie {
	c.domain = domain
	return c
}

// Expires returns the cookie expiry time.
func (c *cookie) Expires() time.Time {
	return c.expires
}

// SetExpires sets the cookie expiry time.
func (c *cookie) SetExpires(expires time.Time) *cookie {
	c.expires = expires
	return c
}

// Secure indicates if cookie is Secure.
func (c *cookie) Secure() bool {
	return c.secure
}

// SetSecure sets the cookie as Secure.
func (c *cookie) SetSecure(secure bool) *cookie {
	c.secure = secure
	return c
}

// HTTPOnly indicates if cookie is HTTPOnly.
func (c *cookie) HTTPOnly() bool {
	return c.httpOnly
}

// SetHTTPOnly sets the cookie as HTTPOnly.
func (c *cookie) SetHTTPOnly(httpOnly bool) *cookie {
	c.httpOnly = httpOnly
	return c
}
