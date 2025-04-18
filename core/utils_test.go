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
		name   string
		pc     *ProvableChain
		target any
		err    error
	}{
		{
			name:   "ProvableChain has target chain pointer directly",
			pc:     NewProvableChain(&wantChain, nil),
			target: &testChain{},
			err:    nil,
		},
		{
			name:   "ProvableChain has target chain directly",
			pc:     NewProvableChain(wantChain, nil),
			target: testChain{},
			err:    nil,
		},
		{
			name:   "ProvableChain has wrapped target chain pointer",
			pc:     NewProvableChain(wrapperChain{&wantChain}, nil),
			target: &testChain{},
			err:    nil,
		},
		{
			name:   "ProvableChain has wrapped target chain",
			pc:     NewProvableChain(&wrapperChain{wantChain}, nil),
			target: testChain{},
			err:    nil,
		},
		{
			name:   "ProvableChain does not have target chain (different struct)",
			pc:     NewProvableChain(wrapperChain{&anotherTestChain{}}, nil),
			target: &testChain{},
			err:    errors.New("chain does not contain *core.testChain; the actual type is *core.ProvableChain{Chain:core.wrapperChain{Chain:*core.anotherTestChain}}"),
		},
		{
			name:   "ProvableChain does not have target chain (target is struct but not pointer)",
			pc:     NewProvableChain(wrapperChain{&testChain{}}, nil),
			target: testChain{},
			err:    errors.New("chain does not contain core.testChain; the actual type is *core.ProvableChain{Chain:core.wrapperChain{Chain:*core.testChain}}"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			initialized := false
			switch c := tt.target.(type) {
			case testChain:
				err = AsChain(tt.pc, &c)
				initialized = c.initialized
			case *testChain:
				err = AsChain(tt.pc, &c)
				initialized = c.initialized
				if err == nil && c != &wantChain {
					t.Errorf("target has an unexpected address")
				}
			}
			if err != tt.err && (err == nil || tt.err == nil || err.Error() != tt.err.Error()) {
				t.Errorf("AsChain() = %v, want %v", err, tt.err)
			}

			wantInitialized := tt.err == nil
			if initialized != wantInitialized {
				t.Errorf("initialized = %v, want %v", initialized, wantInitialized)
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
		name   string
		pc     *ProvableChain
		target any
		err    error
	}{
		{
			name:   "ProvableChain has target prover pointer directly",
			pc:     NewProvableChain(nil, &wantProver),
			target: &testProver{},
			err:    nil,
		},
		{
			name:   "ProvableChain has target prover directly",
			pc:     NewProvableChain(nil, wantProver),
			target: testProver{},
			err:    nil,
		},
		{
			name:   "ProvableChain has wrapped target prover pointer",
			pc:     NewProvableChain(nil, wrapperProver{&wantProver}),
			target: &testProver{},
			err:    nil,
		},
		{
			name:   "ProvableChain has wrapped target prover",
			pc:     NewProvableChain(nil, &wrapperProver{wantProver}),
			target: testProver{},
			err:    nil,
		},
		{
			name:   "ProvableChain does not have target prover",
			pc:     NewProvableChain(nil, wrapperProver{&anotherTestProver{}}),
			target: &testProver{},
			err:    errors.New("prover does not contain *core.testProver; the actual type is *core.ProvableChain{Prover:core.wrapperProver{Prover:*core.anotherTestProver}}"),
		},
		{
			name:   "ProvableChain does not have target prover (target is struct but not pointer)",
			pc:     NewProvableChain(nil, wrapperProver{&testProver{}}),
			target: testProver{},
			err:    errors.New("prover does not contain core.testProver; the actual type is *core.ProvableChain{Prover:core.wrapperProver{Prover:*core.testProver}}"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			initialized := false
			switch p := tt.target.(type) {
			case testProver:
				err = AsProver(tt.pc, &p)
				initialized = p.initialized
			case *testProver:
				err = AsProver(tt.pc, &p)
				initialized = p.initialized
				if err == nil && p != &wantProver {
					t.Errorf("target has unexpected address")
				}
			}
			if err != tt.err && (err == nil || tt.err == nil || err.Error() != tt.err.Error()) {
				t.Errorf("AsProver() = %v, want %v", err, tt.err)
			}

			wantInitialized := tt.err == nil
			if initialized != wantInitialized {
				t.Errorf("initialized = %v, want %v", initialized, wantInitialized)
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
