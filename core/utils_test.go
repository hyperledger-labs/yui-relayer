package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAsChain(t *testing.T) {
	type testChain struct {
		Chain
		initialized bool
	}
	type anotherTestChain struct {
		Chain
	}
	type wrapperChain struct {
		Chain
	}

	wantChain := testChain{
		initialized: true,
	}

	tests := []struct {
		name string
		pc   *ProvableChain
		want bool
	}{
		{
			name: "ProvableChain has target chain pointer directly",
			pc:   NewProvableChain(&wantChain, nil),
			want: true,
		},
		{
			name: "ProvableChain has target chain directly",
			pc:   NewProvableChain(wantChain, nil),
			want: true,
		},
		{
			name: "ProvableChain has wrapped target chain pointer",
			pc:   NewProvableChain(wrapperChain{&wantChain}, nil),
			want: true,
		},
		{
			name: "ProvableChain has wrapped target chain",
			pc:   NewProvableChain(&wrapperChain{wantChain}, nil),
			want: true,
		},
		{
			name: "ProvableChain does not have target chain",
			pc:   NewProvableChain(wrapperChain{&anotherTestChain{}}, nil),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tc testChain
			if got := AsChain(tt.pc, &tc); got != tt.want {
				t.Errorf("AsChain() = %v, want %v", got, tt.want)
			}
			if tc.initialized != tt.want {
				t.Errorf("tc.initialized = %v, want %v", tc.initialized, tt.want)
			}
		})
	}
}

func TestAsProver(t *testing.T) {
	type testProver struct {
		Prover
		initialized bool
	}
	type anotherTestProver struct {
		Prover
	}
	type wrapperProver struct {
		Prover
	}

	wantProver := testProver{
		initialized: true,
	}
	tests := []struct {
		name string
		pc   *ProvableChain
		want bool
	}{
		{
			name: "ProvableChain has target prover pointer directly",
			pc:   NewProvableChain(nil, &wantProver),
			want: true,
		},
		{
			name: "ProvableChain has target prover directly",
			pc:   NewProvableChain(nil, wantProver),
			want: true,
		},
		{
			name: "ProvableChain has wrapped target prover pointer",
			pc:   NewProvableChain(nil, wrapperProver{&wantProver}),
			want: true,
		},
		{
			name: "ProvableChain has wrapped target prover",
			pc:   NewProvableChain(nil, &wrapperProver{wantProver}),
			want: true,
		},
		{
			name: "ProvableChain does not have target prover",
			pc:   NewProvableChain(nil, wrapperProver{&anotherTestProver{}}),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tp testProver
			if got := AsProver(tt.pc, &tp); got != tt.want {
				t.Errorf("AsProver() = %v, want %v", got, tt.want)
			}
			if tp.initialized != tt.want {
				t.Errorf("tp.initialized = %v, want %v", tp.initialized, tt.want)
			}
		})
	}
}

func TestRunUntilComplete(t *testing.T) {
	runtimeError := errors.New("runtime error")

	tests := []struct {
		name    string
		fn      func(int) (bool, error)
		attempt int
		cancel  bool
		err     error
	}{
		{
			name: "Complete immediately",
			fn: func(_ int) (bool, error) {
				return true, nil
			},
			attempt: 1,
			cancel:  false,
			err:     nil,
		},
		{
			name: "Complete on the second try",
			fn: func(attempt int) (bool, error) {
				if attempt == 2 {
					return true, nil
				} else {
					return false, nil
				}
			},
			attempt: 2,
			cancel:  false,
			err:     nil,
		},
		{
			name: "Complete on the third try",
			fn: func(attempt int) (bool, error) {
				if attempt == 3 {
					return true, nil
				} else {
					return false, nil
				}
			},
			attempt: 3,
			cancel:  false,
			err:     nil,
		},
		{
			name: "Error immediately",
			fn: func(_ int) (bool, error) {
				return false, runtimeError
			},
			attempt: 1,
			cancel:  false,
			err:     runtimeError,
		},
		{
			name: "Error on the second try",
			fn: func(attempt int) (bool, error) {
				if attempt == 2 {
					return false, runtimeError
				} else {
					return false, nil
				}
			},
			attempt: 2,
			cancel:  false,
			err:     runtimeError,
		},
		{
			name: "Error immediately with complete true",
			fn: func(_ int) (bool, error) {
				return true, runtimeError
			},
			attempt: 1,
			cancel:  false,
			err:     runtimeError,
		},
		{
			name: "Cancelled",
			fn: func(_ int) (bool, error) {
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
				return tt.fn(attempt)
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
