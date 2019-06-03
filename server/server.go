//
// server.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"

	// self import
	"wgo/daemon"
	"wgo/listener"
)

type (
	Server struct {
		lock       sync.Mutex
		cfg        Config
		tlsConfig  *tls.Config // optional TLS config, used by ServeTLS and ListenAndServeTLS
		listener   *listener.Listener
		engine     Engine
		engine_gen EngineFactory
		mux_gen    MuxFactory
	}
	Config struct {
		Name       string   `mapstructure:"name"`
		Mode       string   `mapstructure:"mode"`
		Engine     string   `mapstructure:"engine"`
		Addr       string   `mapstructure:"addr"`
		Hosts      []string `mapstructure:"hosts"`
		NoAutocert bool     `mapstructure:"no_autocert"`
		NoCallback bool     `mapstructure:"no_callback"`
		CertFile   string   `mapstructure:"cert_file"`
		KeyFile    string   `mapstructure:"key_file"`
	}
)

// NewServer
func NewServer(cfg Config) *Server {
	if cfg.Name == "" {
		cfg.Name = cfg.Mode
	}
	s := &Server{cfg: cfg}
	return s
}

// prepare
func (s *Server) Prepare() {
	if s.Mux() != nil {
		s.Mux().Prepare()
	}
}

// factory
func (s *Server) Factory(fs ...interface{}) *Server {
	// Info("[server.Factory]factories: %d", len(fs))
	if len(fs) > 0 {
		if ef, ok := fs[0].(EngineFactory); ok {
			s.engine_gen = ef
		} else {
			Error("[server.Factory]not found >>EngineFactory<<")
		}
	}
	if len(fs) > 1 {
		if mf, ok := fs[1].(MuxFactory); ok {
			s.mux_gen = mf
		} else {
			Error("[server.Factory]not found >>MuxFactory<<")
		}
	}
	return s
}

// BuildEngine
func (s *Server) BuildEngine() *Server {
	if s.engine_gen != nil {
		eng := s.engine_gen()
		s.SetEngine(eng)
	}
	return s.buildMux()
}

// set engine
func (s *Server) SetEngine(eng Engine) *Server {
	s.engine = eng
	return s
}

// new mux
func (s *Server) buildMux() *Server {
	if s.mux_gen != nil {
		m := s.mux_gen()
		m.SetLogger(logger)
		// m.SetEngine(s.engine)
		s.Engine().SetMux(m)
		Debug("[buildMux]mux_gen")
	}
	return s
}

/* {{{ func (s *Server) Listener() net.Listener
 * Listener returns the net.Listener which this server (is) listening to
 */
func (s *Server) Listener() *listener.Listener {
	return s.listener
}

/* }}} */

/* {{{ func (s *Server) Mux() string
 * Mux returns the mux of the server
 */
func (s *Server) Mux() Mux {
	// Debug("[server.Mux]")
	if eng := s.Engine(); eng != nil {
		return eng.Mux()
	}
	return nil
}

/* }}} */

/* {{{ func (s *Server) Name() string
 * Name returns the name of the server
 */
func (s *Server) Name() string {
	switch {
	case s.cfg.Name != "":
		return s.cfg.Name
	case s.cfg.Mode != "": // 如果没有定义server name, 采用mode
		return s.cfg.Mode
	default:
		return ""
	}
}

/* }}} */

/* {{{ func (s *Server) EngineName() string
* Scheme returns http or https if SSL is enabled
 */
func (s *Server) EngineName() string {
	if s.cfg.Engine != "" {
		return strings.ToLower(s.cfg.Engine)
	}
	return "default"
}

/* }}} */

/* {{{ func (s *Server) Addr() string
 * Addr returns the addr for the server
 */
func (s *Server) Addr() string {
	return s.cfg.Addr
}

/* }}} */

/* {{{ func (s *Server) Port() int
 * Port returns the port which server listening for
 * if no port given with the ListeningAddr, it returns 80
 */
func (s *Server) Port() int {
	a := s.Addr()
	if portIdx := strings.IndexByte(a, ':'); portIdx != -1 {
		p, err := strconv.Atoi(a[portIdx+1:])
		if err != nil {
			return 80
		} else {
			return p
		}
	}
	return 80
}

/* }}} */

/* {{{ func (s *Server) Mode() string
* Scheme returns http or https if SSL is enabled
 */
func (s *Server) Mode() string {
	switch {
	case s.cfg.Mode != "":
		return strings.ToLower(s.cfg.Mode)
	default:
		return MODE_HTTP
	}
}

/* }}} */

/* {{{ func (s *Server) Engine() Engine
* Scheme returns http or https if SSL is enabled
 */
func (s *Server) Engine() Engine {
	return s.engine
}

/* }}} */

/* {{{ func (s *Server) IsListening() bool
 * IsListening returns true if server is listening/started, otherwise false
 */
func (s *Server) IsListening() bool {
	return s != nil && s.listener != nil && s.listener.Addr().String() != ""
}

/* }}} */

/* {{{ func (s *Server) Close() error
 *
 */
func (s *Server) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.IsListening() {
		return fmt.Errorf("server is closed")
	}
	return s.listener.Close()
}

/* }}} */

/* {{{ func (s *Server) IsIdle() bool
 * listener没有请求, 代表服务器空闲
 */
func (s *Server) IsIdle() bool {
	s.listener.Wait()
	return true
}

/* }}} */

/* {{{ func (s *Server) ListenAndServe(d *daemon.Daemon) error
 *
 */
func (s *Server) ListenAndServe(d *daemon.Daemon) (err error) {
	s.lock.Lock()
	if s.IsListening() {
		Info("%s already listening", s.Addr())
		return errors.New("already listening")
	}

	var nl net.Listener
	if nl, err = d.GetListener(s.cfg.Addr); err == nil {
		//Info("Get listener from daemon pool, addr: %s", s.cfg.Addr)
	} else {
		if nl, err = net.Listen("tcp4", s.cfg.Addr); err != nil {
			//Info("Create listener failed: %s, addr: %s", err, s.cfg.Addr)
			return
		} else {
			//Info("Create listener and add to daemon pool, addr: %s", s.cfg.Addr)
			s.listener = listener.WrapListener(nl)
			d.AddListener(nl) // 把listener加入daemon, 以利用daemon的Reload
		}
		// tls config
		if s.Mode() == MODE_HTTPS {
			config := &tls.Config{}
			config.NextProtos = append(config.NextProtos, "http/1.1")
			config.PreferServerCipherSuites = true
			if !s.cfg.NoAutocert && s.cfg.CertFile == "" && s.cfg.KeyFile == "" {
				// Let's Encrypt
				cacheDir := "wgo-autocert"
				if err = os.MkdirAll(cacheDir, 0700); err != nil {
					Error("cannot create -autocertCacheDir=%q: %s", cacheDir, err)
				}
				Debug("autocert hosts: %s", s.cfg.Hosts)
				manager := &autocert.Manager{
					Prompt:     autocert.AcceptTOS,
					HostPolicy: autocert.HostWhitelist(s.cfg.Hosts...),
					Cache:      autocert.DirCache("wgo-autocert"),
				}
				config.GetCertificate = manager.GetCertificate
				if !s.cfg.NoCallback {
					// for Let's Encrypt callbacks over http
					// 80端口不能被占用(Let's Encrypt callbacks over http)
					mux := &http.ServeMux{}
					mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
						newURI := "https://" + r.Host + r.URL.String()
						http.Redirect(w, r, newURI, http.StatusFound)
					})
					httpSrv := &http.Server{
						ReadTimeout:  5 * time.Second,
						WriteTimeout: 5 * time.Second,
						IdleTimeout:  120 * time.Second,
						Handler:      manager.HTTPHandler(mux),
						Addr:         ":http", // 必须是80
					}
					go func() {
						Debug("Starting HTTP server on %s, for Encrypt callbacks", httpSrv.Addr)
						err := httpSrv.ListenAndServe()
						if err != nil {
							Info("httpsSrv.ListenAndServe() failed with %s", err)
						}
					}()
				}
			}
			if config.GetCertificate == nil || (s.cfg.CertFile != "" && s.cfg.KeyFile != "") { // 提供了key file
				config.Certificates = make([]tls.Certificate, 1)
				if config.Certificates[0], err = tls.LoadX509KeyPair(s.cfg.CertFile, s.cfg.KeyFile); err != nil {
					Error("LoadX509KeyPair error: %s", err)
					return
				}
			}
			s.tlsConfig = config
		}
	}
	s.lock.Unlock()

	if s.tlsConfig != nil {
		// https需要在普通listener上再包一层
		Info("Starting %s(https:%d)", s.Name(), s.Port())
		return s.Engine().Start(tls.NewListener(s.listener, s.tlsConfig))
	} else {
		Info("Starting %s(tcp:%d)", s.Name(), s.Port())
		return s.Engine().Start(s.listener)
	}
}

/* }}} */
