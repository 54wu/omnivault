package api

import (
	"context"
	"net"
	"net/http"
	"time"
)

// PipeName is the named pipe that the vault HTTP server listens on. Named
// pipes ride the SMB/LPC kernel transport rather than the TCP/IP stack, so
// they are NOT subject to the network-filter drivers (e.g. Tencent TAO) that
// block loopback TCP connections from Go processes.
const PipeName = `\\.\pipe\omnivault`

// pipeListener wraps a winio pipe listener so http.Server can serve over it.
type pipeListener struct {
	net.Listener
}

// StartPipe begins serving HTTP over the named pipe. It returns immediately
// (Serve runs in a goroutine). Error is returned if the pipe cannot be bound.
func (s *Server) StartPipe() (net.Listener, error) {
	ln, err := listenPipe(PipeName)
	if err != nil {
		return nil, err
	}
	go s.server.Serve(ln)
	return ln, nil
}

// pipeDialContext returns a DialContext that connects to the vault named pipe.
// Used by the HTTP client transport so every request rides the pipe instead of
// loopback TCP.
func pipeDialContext(_ context.Context, _, _ string) (net.Conn, error) {
	return dialPipeTimeout(PipeName, 5*time.Second)
}

// httpClient returns an http.Client whose transport dials the named pipe.
func pipeClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: pipeDialContext,
		},
	}
}