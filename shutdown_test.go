package shutdown_test

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/dkotTech/shutdown"
)

func ExampleGraceful() {
	wait := shutdown.Graceful(context.Background(), map[string]shutdown.Operation{
		"postgres": func(ctx context.Context) error {
			fmt.Println("postgres Closed()")

			return nil
		},
		"clickhouse": func(ctx context.Context) error {
			fmt.Println("clickhouse Closed()")
			time.Sleep(time.Second * 3)
			return fmt.Errorf("clickhouse error")
		},
	},
		shutdown.WithOsSignals([]os.Signal{syscall.SIGINT}),
		shutdown.WithTimeout(time.Second),
		shutdown.WithTimeoutFunc(func() {
			fmt.Println("timeout has been elapsed, force exit")
			os.Exit(0)
		}),
		shutdown.WithOnCleaningUpFailedFunc(func(name string, err error) {
			fmt.Println(err)
		}))

	<-wait
}
