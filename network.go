package main

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	proxyTypeGithub = "GH-PROXY"
	proxyTypeHTTP   = "HTTP(S)"
	proxyTypeSOCKS5 = "SOCKS5"
)

func newHTTPClientWithProxy(sd *SettingsData, timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:                 GetProxyURL(),
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
		ForceAttemptHTTP2:     true,
	}

	if sd != nil {
		if proxyURL, ok := proxyURLFromSettings(getString(sd.proxyType), getString(sd.ghProxy)); ok {
			transport.Proxy = http.ProxyURL(proxyURL)
			logger.Infof("using proxy: %s", proxyURL.String())
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

func rewriteWithGithubProxy(sd *SettingsData, reqURL string) string {
	if sd == nil {
		return reqURL
	}
	proxy := strings.TrimSpace(getString(sd.ghProxy))
	if isGithubProxyPrefix(getString(sd.proxyType), proxy) {
		return pathJoin(proxy, reqURL)
	}
	return reqURL
}

func proxyURLFromSettings(proxyType, proxy string) (*url.URL, bool) {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" || isGithubProxyPrefix(proxyType, proxy) {
		return nil, false
	}

	if !strings.Contains(proxy, "://") {
		switch proxyType {
		case proxyTypeSOCKS5:
			proxy = "socks5://" + proxy
		default:
			proxy = "http://" + proxy
		}
	}

	parsed, err := url.Parse(proxy)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		logger.Warnf("proxy URL parse failed: %s, err=%v", proxy, err)
		return nil, false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5":
		return parsed, true
	default:
		logger.Warnf("unsupported proxy scheme: %s", parsed.Scheme)
		return nil, false
	}
}

func isGithubProxyPrefix(proxyType, proxy string) bool {
	if proxyType != proxyTypeGithub {
		return false
	}
	parsed, err := url.Parse(strings.TrimSpace(proxy))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return false
	}
	return strings.HasPrefix(strings.ToLower(parsed.Scheme), "http")
}
