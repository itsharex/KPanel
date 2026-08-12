package browsercore

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	defaultConnectTimeout         = 5 * time.Second
	defaultTLSHandshakeTimeout    = 10 * time.Second
	defaultResponseHeaderTimeout  = 15 * time.Second
	defaultIdleConnectionTimeout  = 30 * time.Second
	defaultMaxResponseHeaderBytes = 64 << 10
)

type SafeDialer struct {
	Policy *TargetPolicy
	Dialer net.Dialer
}

func NewSafeDialer(policy *TargetPolicy) *SafeDialer {
	return &SafeDialer{
		Policy: policy,
		Dialer: net.Dialer{Timeout: defaultConnectTimeout, KeepAlive: 30 * time.Second},
	}
}

func (d *SafeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d.Policy == nil {
		return nil, errors.New("browser target policy is required")
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid dial address: %w", err)
	}
	resolveContext, cancelResolve := context.WithTimeout(ctx, defaultConnectTimeout)
	target, err := d.Policy.Resolve(resolveContext, "http://"+net.JoinHostPort(host, port)+"/")
	cancelResolve()
	if err != nil {
		return nil, err
	}
	var failures []error
	for _, candidate := range target.Addresses {
		conn, dialErr := d.Dialer.DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		failures = append(failures, dialErr)
	}
	return nil, errors.Join(failures...)
}

func NewSafeTransport(policy *TargetPolicy) *http.Transport {
	dialer := NewSafeDialer(policy)
	return &http.Transport{
		Proxy:                  nil,
		DialContext:            dialer.DialContext,
		ForceAttemptHTTP2:      true,
		DisableCompression:     true,
		MaxIdleConns:           128,
		MaxIdleConnsPerHost:    6,
		MaxConnsPerHost:        6,
		IdleConnTimeout:        defaultIdleConnectionTimeout,
		TLSHandshakeTimeout:    defaultTLSHandshakeTimeout,
		ResponseHeaderTimeout:  defaultResponseHeaderTimeout,
		ExpectContinueTimeout:  time.Second,
		MaxResponseHeaderBytes: defaultMaxResponseHeaderBytes,
	}
}
