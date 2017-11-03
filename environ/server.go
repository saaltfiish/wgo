package environ

import (
	"strings"
	"wgo/server"
)

const (
	defaultListenHost = "0.0.0.0"
	defaultListenPort = "9999"
	defaultServerName = "WGO"
	defaultServerMode = "http"
)

var (
	scs []server.Config
)

/* {{{ func ServersConfig(cfg *Config) []Server
 *
 */
func ServersConfig(cfg *Config) []server.Config {
	if cfg.Get(CFG_KEY_SERVERS) != nil {
		scs = []server.Config{}
		if err := cfg.UnmarshalKey(CFG_KEY_SERVERS, &scs); err != nil {
			panic(err)
		}
		for i := 0; i < len(scs); i++ {
			if addr := scs[i].Addr; addr != "" {
				if portIdx := strings.IndexByte(addr, ':'); portIdx == 0 {
					// if contains only :port	,then the : is the first letter, so we dont have setted a hostname, lets set it
					scs[i].Addr = defaultListenHost + addr
				}
				if portIdx := strings.IndexByte(addr, ':'); portIdx < 0 {
					// missing port part, add it
					scs[i].Addr = addr + ":80"
				}
			}
		}
	} else {
		scs = DefaultServers(cfg)
	}
	return scs
}

/* }}} */

/* {{{ func DefaultServers(cfg *Cofnig) []Server
 *
 */
func DefaultServers(cfg *Config) []server.Config {
	// find hsotname:port
	addr := ""
	if l := cfg.String(CFG_KEY_LISTEN); l != "" {
		addr = l
	} else if a := cfg.String(CFG_KEY_ADDR); a != "" {
		addr = a
	} else if p := cfg.String(CFG_KEY_PORT); p != "" {
		addr = ":" + p
	}
	if addr != "" {
		if portIdx := strings.IndexByte(addr, ':'); portIdx == 0 {
			// if contains only :port	,then the : is the first letter, so we dont have setted a hostname, lets set it
			addr = defaultListenHost + addr
		}
		if portIdx := strings.IndexByte(addr, ':'); portIdx < 0 {
			// missing port part, add it
			addr = addr + ":80"
		}
		sc := server.Config{
			Name: defaultServerName,
			Mode: defaultServerMode,
			Addr: addr,
		}
		return []server.Config{sc}
	}
	return nil
}

/* }}} */
