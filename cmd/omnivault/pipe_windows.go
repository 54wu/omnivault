//go:build windows

package main

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Microsoft/go-winio"
)

// vaultPipeName matches internal/api.PipeName. Requests between the CLI and
// the serve child ride this named pipe so they bypass TCP loopback, which is
// blocked by network-filter drivers (e.g. Tencent TAO) on some machines.
const vaultPipeName = `\\.\pipe\omnivault`

func pipeDialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, vaultPipeName)
}

// pipeHTTPClient returns an http.Client whose transport dials the named pipe.
func pipeHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: pipeDialContext,
		},
	}
}