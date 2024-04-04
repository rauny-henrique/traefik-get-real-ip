package traefik_get_real_ip

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"runtime/debug"
)

const (
	xRealIP       = "X-Real-Ip"
	xForwardedFor = "X-Forwarded-For"
	remoteAddr 	  = "RemoteAddr"
)

type Proxy struct {
	ProxyHeadername  string `yaml:"proxyHeadername"`
	ProxyHeadervalue string `yaml:"proxyHeadervalue"`
	RealIP           string `yaml:"realIP"`
	OverwriteXFF     bool   `yaml:"overwriteXFF"` // override X-Forwarded-For
	OverwriteRA      bool   `yaml:"overwriteRA"` //  override RemoteAddr
}

// Config the plugin configuration.
type Config struct {
	Proxy []Proxy `yaml:"proxy"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// GetRealIP Define plugin
type GetRealIP struct {
	next  http.Handler
	name  string
	proxy []Proxy
}

// New creates and returns a new realip plugin instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	log("☃️ Config loaded.(%d) %v", len(config.Proxy), config)

	return &GetRealIP{
		next:  next,
		name:  name,
		proxy: config.Proxy,
	}, nil
}

func (g *GetRealIP) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		if panicInfo := recover(); panicInfo != nil {
			log("%v, %s", panicInfo, string(debug.Stack()))
			g.next.ServeHTTP(rw, req)
		}
	}()

	// fmt.Println("☃️：", g.proxy, "remoteaddr", req.RemoteAddr)
	var realIPStr string
	for _, proxy := range g.proxy {
		if proxy.ProxyHeadername == "*" || req.Header.Get(proxy.ProxyHeadername) == proxy.ProxyHeadervalue {
			// log("🐸 Current Proxy：%s(%s)", proxy.ProxyHeadervalue, proxy.ProxyHeadername)

			// CDN
			nIP := req.Header.Get(proxy.RealIP)
			if proxy.RealIP == remoteAddr {
				nIP, _, _ = net.SplitHostPort(req.RemoteAddr)
			}
			forwardedIPs := strings.Split(nIP, ",")

			// log("👀 IPs:'%v' %d", forwardedIPs, len(forwardedIPs))
			for i := 0; i <= len(forwardedIPs)-1; i++ {
				trimmedIP := strings.TrimSpace(forwardedIPs[i])
				finalIP := g.getIP(trimmedIP)
				// log("currentIP:%s, index:%d, result:%s", trimmedIP, i, finalIP)
				if finalIP != nil {
					realIPStr = finalIP.String()
					break
				}
			}
		}
		// realIP
		if realIPStr != "" {
			if proxy.OverwriteXFF {
				// log("🐸 Modify XFF to:%s", realIPStr)
				req.Header.Set(xForwardedFor, realIPStr)
			}
			if proxy.OverwriteRA {
				// log("🐸 Modify RemoteAddr to:%s", realIPStr)
				req.Header.Set(remoteAddr, realIPStr)
			}
			req.Header.Set(xRealIP, realIPStr)
			break
		}
	}
	g.next.ServeHTTP(rw, req)
}

// getIP is used to obtain valid IP addresses. The parameter s is the input IP text,
// which should be in the format of x.x.x.x or x.x.x.x:1234.
func (g *GetRealIP) getIP(s string) net.IP {
	pureIP, _, err := net.SplitHostPort(s)
	if err != nil {
		pureIP = s
	}
	ip := net.ParseIP(pureIP)
	return ip
}

// log is used for logging output, with a usage similar to Sprintf,
// but it already includes a newline character at the end.
func log(format string, a ...interface{}) {
	os.Stdout.WriteString("[get-realip] " + fmt.Sprintf(format, a...) + "\n")
}

// err is used for output err logs, and it usage is simillar to Sprintf,
// but with a newline character already included at the end.
// func err(format string, a ...interface{}) {
// 	os.Stderr.WriteString(fmt.Sprintf(format, a...) + "\n")
// }
