package output

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/huh/spinner"
)

// SpinnerOption configures a spinner.
type SpinnerOption func(*spinnerConfig)

type spinnerConfig struct {
	title   string
	timeout time.Duration
}

// WithTitle sets the spinner title.
func WithTitle(title string) SpinnerOption {
	return func(c *spinnerConfig) {
		c.title = title
	}
}

// WithTimeout sets the spinner timeout.
func WithTimeout(timeout time.Duration) SpinnerOption {
	return func(c *spinnerConfig) {
		c.timeout = timeout
	}
}

// RunWithSpinner executes an action with a spinner.
// Returns the action's error if any.
func RunWithSpinner(ctx context.Context, action func() error, opts ...SpinnerOption) error {
	cfg := &spinnerConfig{
		title:   "Working...",
		timeout: 0,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// If not a TTY, just run the action directly
	if !IsTTY() {
		return action()
	}

	// Create context with timeout if specified
	actionCtx := ctx
	var cancel context.CancelFunc
	if cfg.timeout > 0 {
		actionCtx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	// Create error channel
	errCh := make(chan error, 1)

	// Run action in background
	go func() {
		errCh <- action()
	}()

	// Create and run spinner
	s := spinner.New().Title(cfg.title)

	// Run spinner with action
	spinnerErr := s.Action(func() {
		select {
		case <-actionCtx.Done():
			return
		case <-errCh:
			return
		}
	}).Run()

	if spinnerErr != nil {
		return fmt.Errorf("spinner error: %w", spinnerErr)
	}

	// Wait for action result
	select {
	case err := <-errCh:
		return err
	case <-actionCtx.Done():
		return actionCtx.Err()
	}
}

// WaitWithSpinner waits for a condition with a spinner.
func WaitWithSpinner(ctx context.Context, title string, condition func() (bool, error), interval time.Duration) error {
	return RunWithSpinner(ctx, func() error {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				done, err := condition()
				if err != nil {
					return err
				}
				if done {
					return nil
				}
			}
		}
	}, WithTitle(title))
}
