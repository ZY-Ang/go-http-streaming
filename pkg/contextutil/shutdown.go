package contextutil

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type InterruptHandler func(sig os.Signal)

// WithShutdown returns a context which will be done on (syscall.SIGINT, syscall.SIGTERM).
// Optional handlers can be provided which will be invoked on cancellation.
func WithShutdown(super context.Context, optionalHandlers ...InterruptHandler) context.Context {
	ctx, cancel := context.WithCancel(super)
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		var receivedSignal os.Signal
		select {
		case receivedSignal = <-s:
			// case if received stop signal regardless of parent
		case <-ctx.Done():
			// case if parent context canceled - child no longer needs to bother with stop signal
		}
		cancel()
		for _, handler := range optionalHandlers {
			if handler != nil {
				handler(receivedSignal)
			}
		}
		signal.Stop(s)
		close(s)
	}()
	return ctx
}
