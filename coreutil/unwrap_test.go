package coreutil_test

import (
	"errors"
	"testing"

	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/coreutil"
	"github.com/hyperledger-labs/yui-relayer/otelcore"
)

func TestUnwrapChain(t *testing.T) {
	type testChain struct {
		core.Chain
		initialized bool
	}
	type anotherTestChain struct {
		core.Chain
	}
	type wrapperChain struct {
		core.Chain
	}

	wantChain := testChain{
		initialized: true,
	}

	tests := []struct {
		name   string
		pc     *core.ProvableChain
		target any
		err    error
	}{
		{
			name:   "ProvableChain has target chain pointer directly",
			pc:     core.NewProvableChain(&wantChain, nil),
			target: &testChain{},
			err:    nil,
		},
		{
			name:   "ProvableChain has target chain directly",
			pc:     core.NewProvableChain(wantChain, nil),
			target: testChain{},
			err:    nil,
		},
		{
			name:   "ProvableChain has wrapped target chain pointer",
			pc:     core.NewProvableChain(otelcore.NewChain(&wantChain, nil), nil),
			target: &testChain{},
			err:    nil,
		},
		{
			name:   "ProvableChain has wrapped target chain",
			pc:     core.NewProvableChain(otelcore.NewChain(wantChain, nil), nil),
			target: testChain{},
			err:    nil,
		},
		{
			name:   "ProvableChain does not have target chain (different struct)",
			pc:     core.NewProvableChain(otelcore.NewChain(anotherTestChain{}, nil), nil),
			target: testChain{},
			err:    errors.New("failed to unwrap chain: expected=coreutil_test.testChain, actual=coreutil_test.anotherTestChain"),
		},
		{
			name:   "ProvableChain has a target chain wrapped by an unknown chain",
			pc:     core.NewProvableChain(wrapperChain{wantChain}, nil),
			target: testChain{},
			err:    errors.New("failed to unwrap chain: expected=coreutil_test.testChain, actual=coreutil_test.wrapperChain"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			switch c := tt.target.(type) {
			case testChain:
				c, err = coreutil.UnwrapChain[testChain](tt.pc)
				if err == nil && c != wantChain {
					t.Errorf("c = %v, want %v", c, wantChain)
				}
			case *testChain:
				c, err = coreutil.UnwrapChain[*testChain](tt.pc)
				if err == nil && c != &wantChain {
					t.Errorf("unwrapped chain has an unexpected address")
				}
			}
			if err != tt.err && (err == nil || tt.err == nil || err.Error() != tt.err.Error()) {
				t.Errorf("err = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestUnwrapProver(t *testing.T) {
	type testProver struct {
		core.Prover
		initialized bool
	}
	type anotherTestProver struct {
		core.Prover
	}
	type wrapperProver struct {
		core.Prover
	}

	wantProver := testProver{
		initialized: true,
	}
	tests := []struct {
		name   string
		pc     *core.ProvableChain
		target any
		err    error
	}{
		{
			name:   "ProvableChain has target prover pointer directly",
			pc:     core.NewProvableChain(nil, &wantProver),
			target: &testProver{},
			err:    nil,
		},
		{
			name:   "ProvableChain has target prover directly",
			pc:     core.NewProvableChain(nil, wantProver),
			target: testProver{},
			err:    nil,
		},
		{
			name:   "ProvableChain has wrapped target prover pointer",
			pc:     core.NewProvableChain(nil, otelcore.NewProver(&wantProver, "", nil)),
			target: &testProver{},
			err:    nil,
		},
		{
			name:   "ProvableChain has wrapped target prover",
			pc:     core.NewProvableChain(nil, otelcore.NewProver(wantProver, "", nil)),
			target: testProver{},
			err:    nil,
		},
		{
			name:   "ProvableChain does not have target prover (different struct)",
			pc:     core.NewProvableChain(nil, otelcore.NewProver(anotherTestProver{}, "", nil)),
			target: testProver{},
			err:    errors.New("failed to unwrap prover: expected=coreutil_test.testProver, actual=coreutil_test.anotherTestProver"),
		},
		{
			name:   "ProvableChain has a target prover wrapped by an unknown prover",
			pc:     core.NewProvableChain(nil, wrapperProver{wantProver}),
			target: testProver{},
			err:    errors.New("failed to unwrap prover: expected=coreutil_test.testProver, actual=coreutil_test.wrapperProver"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			switch p := tt.target.(type) {
			case testProver:
				p, err = coreutil.UnwrapProver[testProver](tt.pc)
				if err == nil && p != wantProver {
					t.Errorf("p = %v, want %v", p, wantProver)
				}
			case *testProver:
				p, err = coreutil.UnwrapProver[*testProver](tt.pc)
				if err == nil && p != &wantProver {
					t.Errorf("unwrapped prover has an unexpected address")
				}
			}
			if err != tt.err && (err == nil || tt.err == nil || err.Error() != tt.err.Error()) {
				t.Errorf("err = %v, want %v", err, tt.err)
			}
		})
	}
}
