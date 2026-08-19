//go:build !windows

package api

import (
	"context"
	"errors"
	"net"
	"time"
)

// The named-pipe transport is Windows-only. On other platforms these exist so
// the package still compiles; callers that rely on StartPipe/pipeClient will
// receive a clear "not supported" error at runtime. On these platforms the
// server is reached over plain TCP/HTTP instead.

// ErrPipeUnsupported is returned when a named-pipe API is used off Windows.
var ErrPipeUnsupported = errors.New("named-pipe transport is only supported on Windows")

func listenPipe(name string) (net.Listener, error) {
	return nil, ErrPipeUnsupported
}

func dialPipeTimeout(name string, timeout time.Duration) (net.Conn, error) {
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return nil, ErrPipeUnsupported
}