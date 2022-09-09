package util

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

// RunTasks runs all the given tasks concurrently and waits for all of them to completed.
// If one task is canceled, all the other tasks are canceled.
func RunTasks(ctx context.Context, tasks []func(context.Context) error) error {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		wg.Wait()
	}()

	errch := make(chan error, len(tasks))

	runTask := func(task func(context.Context) error) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v\n%s\n", r, string(debug.Stack())) //nolint: revive // newlines are intentional
			}
			wg.Done()
		}()
		return task(ctx)
	}

	wg.Add(len(tasks))
	for _, task := range tasks {
		task := task
		go func() {
			errch <- runTask(task)
		}()
	}

	for complete := 0; complete < len(tasks); {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errch:
			if err != nil {
				return err
			}
			complete++
		}
	}

	return nil
}
