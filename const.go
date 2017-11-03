package wgo

import (
	"wgo/whttp"
)

// wgo
const (
	VERSION = "0.9.4"

	MODE_HTTP  = "http"
	MODE_HTTPS = "https"
	MODE_RPC   = "rpc"
	MODE_GRPC  = "grpc"
	MODE_WRPC  = "wrpc"
	MODE_SS    = "ss"
	MODE_WSS   = "wss"
)

// http status
const (
	StatusContinue           = whttp.StatusContinue
	StatusSwitchingProtocols = whttp.StatusSwitchingProtocols

	StatusOK                   = whttp.StatusOK
	StatusCreated              = whttp.StatusCreated
	StatusAccepted             = whttp.StatusAccepted
	StatusNonAuthoritativeInfo = whttp.StatusNonAuthoritativeInfo
	StatusNoContent            = whttp.StatusNoContent
	StatusResetContent         = whttp.StatusResetContent
	StatusPartialContent       = whttp.StatusPartialContent

	StatusMultipleChoices   = whttp.StatusMultipleChoices
	StatusMovedPermanently  = whttp.StatusMovedPermanently
	StatusFound             = whttp.StatusFound
	StatusSeeOther          = whttp.StatusSeeOther
	StatusNotModified       = whttp.StatusNotModified
	StatusUseProxy          = whttp.StatusUseProxy
	StatusTemporaryRedirect = whttp.StatusTemporaryRedirect

	StatusBadRequest                    = whttp.StatusBadRequest
	StatusUnauthorized                  = whttp.StatusUnauthorized
	StatusPaymentRequired               = whttp.StatusPaymentRequired
	StatusForbidden                     = whttp.StatusForbidden
	StatusNotFound                      = whttp.StatusNotFound
	StatusMethodNotAllowed              = whttp.StatusMethodNotAllowed
	StatusNotAcceptable                 = whttp.StatusNotAcceptable
	StatusProxyAuthRequired             = whttp.StatusProxyAuthRequired
	StatusRequestTimeout                = whttp.StatusRequestTimeout
	StatusConflict                      = whttp.StatusConflict
	StatusGone                          = whttp.StatusGone
	StatusLengthRequired                = whttp.StatusLengthRequired
	StatusPreconditionFailed            = whttp.StatusPreconditionFailed
	StatusRequestEntityTooLarge         = whttp.StatusRequestEntityTooLarge
	StatusRequestURITooLong             = whttp.StatusRequestURITooLong
	StatusUnsupportedMediaType          = whttp.StatusUnsupportedMediaType
	StatusRequestedRangeNotSatisfiable  = whttp.StatusRequestedRangeNotSatisfiable
	StatusExpectationFailed             = whttp.StatusExpectationFailed
	StatusTeapot                        = whttp.StatusTeapot
	StatusPreconditionRequired          = whttp.StatusPreconditionRequired
	StatusTooManyRequests               = whttp.StatusTooManyRequests
	StatusRequestHeaderFieldsTooLarge   = whttp.StatusRequestHeaderFieldsTooLarge
	StatusUnavailableForLegalReasons    = whttp.StatusUnavailableForLegalReasons
	StatusInternalServerError           = whttp.StatusInternalServerError
	StatusNotImplemented                = whttp.StatusNotImplemented
	StatusBadGateway                    = whttp.StatusBadGateway
	StatusServiceUnavailable            = whttp.StatusServiceUnavailable
	StatusGatewayTimeout                = whttp.StatusGatewayTimeout
	StatusHTTPVersionNotSupported       = whttp.StatusHTTPVersionNotSupported
	StatusNetworkAuthenticationRequired = whttp.StatusNetworkAuthenticationRequired
)
