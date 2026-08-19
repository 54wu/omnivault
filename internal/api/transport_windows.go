//go:build windows

package api

import (
	"context"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

func listenPipe(name string) (net.Listener, error) {
	return winio.ListenPipe(name, &winio.PipeConfig{
		SecurityDescriptor: "", // default DACL lets the creating user reconnect
		MessageMode:        false,
		InputBufferSize:    64 * 1024,
		OutputBufferSize:   64 * 1024,
	})
}

func dialPipeTimeout(name string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return winio.DialPipeContext(ctx, name)
}