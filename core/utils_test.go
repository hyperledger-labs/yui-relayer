package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunUntilComplete(t *testing.T) {
	runtimeError := errors.New("runtime error")

	tests := []struct {
		name    string
		fn      func() (bool, error)
		attempt int
		cancel  bool
		err     error
	}{
		{
			name: "Complete immediately",
			fn: func() (bool, error) {
				return true, nil
			},
			attempt: 1,
			cancel:  false,
			err:     nil,
		},
		{
			name: "Complete on the second try",
			fn: func() func() (bool, error) {
				attempt := 0
				return func() (bool, error) {
					attempt++
					if attempt == 2 {
						return true, nil
					} else {
						return false, nil
					}
				}
			}(),
			attempt: 2,
			cancel:  false,
			err:     nil,
		},
		{
			name: "Complete on the third try",
			fn: func() func() (bool, error) {
				attempt := 0
				return func() (bool, error) {
					attempt++
					if attempt == 3 {
						return true, nil
					} else {
						return false, nil
					}
				}
			}(),
			attempt: 3,
			cancel:  false,
			err:     nil,
		},
		{
			name: "Error immediately",
			fn: func() (bool, error) {
				return false, runtimeError
			},
			attempt: 1,
			cancel:  false,
			err:     runtimeError,
		},
		{
			name: "Error on the second try",
			fn: func() func() (bool, error) {
				attempt := 0
				return func() (bool, error) {
					attempt++
					if attempt == 2 {
						return false, runtimeError
					} else {
						return false, nil
					}
				}
			}(),
			attempt: 2,
			cancel:  false,
			err:     runtimeError,
		},
		{
			name: "Error immediately with complete true",
			fn: func() (bool, error) {
				return true, runtimeError
			},
			attempt: 1,
			cancel:  false,
			err:     runtimeError,
		},
		{
			name: "Cancelled",
			fn: func() (bool, error) {
				return false, nil
			},
			attempt: 1,
			cancel:  true,
			err:     context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if tt.cancel {
				cancel()
			} else {
				defer cancel()
			}

			attempt := 0
			fn := func() (bool, error) {
				attempt++
				return tt.fn()
			}
			if err := runUntilComplete(ctx, time.Millisecond, fn); err != tt.err {
				t.Errorf("err = %v; want %v", err, tt.err)
			}
			if attempt != tt.attempt {
				t.Errorf("attempt = %v; want %v", attempt, tt.attempt)
			}
		})
	}
}
