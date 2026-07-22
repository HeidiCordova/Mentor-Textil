package adapters

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	rp *httputil.ReverseProxy
}

func NewProxy(targetURL string) *Proxy {
	u, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid proxy target: " + targetURL)
	}
	rp := httputil.NewSingleHostReverseProxy(u)
	rp.ModifyResponse = func(res *http.Response) error {
		res.Header.Del("Access-Control-Allow-Origin")
		res.Header.Del("Access-Control-Allow-Methods")
		res.Header.Del("Access-Control-Allow-Headers")
		res.Header.Del("Access-Control-Allow-Credentials")
		res.Header.Del("Access-Control-Max-Age")
		res.Header.Del("Vary")
		return nil
	}
	rp.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
	}
	return &Proxy{rp: rp}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.rp.ServeHTTP(w, r)
}
