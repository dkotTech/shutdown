package shutdown

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type shutdownOpts struct {
	osSignals []os.Signal

	timeout     time.Duration
	timeoutFunc func()

	onCleaningUpFailedFunc func(name string, err error)
}

type Option func(*shutdownOpts)

func WithOsSignals(signals []os.Signal) Option {
	return func(opts *shutdownOpts) {
		opts.osSignals = signals
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(opts *shutdownOpts) {
		opts.timeout = timeout
	}
}

func WithTimeoutFunc(timeoutFunc func()) Option {
	return func(opts *shutdownOpts) {
		opts.timeoutFunc = timeoutFunc
	}
}

func WithOnCleaningUpFailedFunc(onCleaningUpFailedFunc func(name string, err error)) Option {
	return func(opts *shutdownOpts) {
		opts.onCleaningUpFailedFunc = onCleaningUpFailedFunc
	}
}

type Operation func(ctx context.Context) error

func Graceful(ctx context.Context, ops map[string]Operation, options ...Option) <-chan struct{} {
	opts := &shutdownOpts{
		osSignals:   []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP},
		timeout:     time.Second * 2,
		timeoutFunc: func() {},
		onCleaningUpFailedFunc: func(name string, err error) {
			slog.Error(err.Error(), "operation", name)
		},
	}

	for _, option := range options {
		option(opts)
	}

	wait := make(chan struct{})

	go func() {
		s := make(chan os.Signal, 1)

		signal.Notify(s, opts.osSignals...)

		<-s

		timeoutFunc := time.AfterFunc(opts.timeout, opts.timeoutFunc)

		defer timeoutFunc.Stop()

		var wg sync.WaitGroup

		for key, op := range ops {
			wg.Add(1)
			innerOp := op
			innerKey := key
			go func() {
				defer wg.Done()

				if err := innerOp(ctx); err != nil {
					opts.onCleaningUpFailedFunc(innerKey, err)
					return
				}
			}()
		}

		wg.Wait()

		close(wait)
	}()

	return wait
}
