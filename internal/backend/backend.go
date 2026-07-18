// Package backend exports the runtime backend type
package backend

import (
	"net/http/httputil"
	"net/url"
)

type Backend struct {
	URL   *url.URL
	Proxy *httputil.ReverseProxy
}

func New(be string) (*Backend, error) {
	target, err := url.Parse(be)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
		},
	}
	return &Backend{
		Proxy: proxy,
		URL:   target,
	}, nil
}
