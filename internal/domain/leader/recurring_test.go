package leader

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/infra/apm/tracing"
)

func TestRecurringTaskRunner_Start(t *testing.T) {
	type fields struct {
		incrementer incrementer
		leaderLock  Lock
	}
	tests := []struct {
		name      string
		fields    fields
		shouldRun bool
	}{
		{
			"should not return anything if the leader lock returns false ",
			fields{
				incrementer: incrementer{},
				leaderLock:  ConstantLock(false),
			},
			false,
		},
		{
			"should run if the leader lock returns true",
			fields{
				incrementer: incrementer{},
				leaderLock:  ConstantLock(true),
			},
			true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tasks := []InternalRecurringFunction{
				NewInternalRecurringFunction(
					"inc",
					10*time.Millisecond,
					func(ctx context.Context, checker Checker) error {
						if checker.IsLeader() {
							tt.fields.incrementer.incr()
						}
						return nil
					},
				),
			}
			// Sanity check. Nothing should run if before Start() is called
			assert.EqualValues(t, 0, tt.fields.incrementer.Value())

			r := NewInternalRecurringFunctionRunner(tasks, tracing.NoopTracer{}, tt.fields.leaderLock)
			r.Start()
			if tt.shouldRun {
				assert.Eventually(t, func() bool {
					return tt.fields.incrementer.Value() > 0
				}, 10*time.Second, 300*time.Millisecond)
			} else {
				assert.EqualValues(t, 0, tt.fields.incrementer.Value())
			}
			r.Stop()
		})
	}
}

type incrementer struct {
	i uint32
}

func (i *incrementer) Value() uint32 {
	return atomic.LoadUint32(&i.i)
}

func (i *incrementer) incr() {
	atomic.AddUint32(&i.i, 1)
}

type ConstantLock bool

func (n ConstantLock) IsLeader() bool {
	return bool(n)
}

func (n ConstantLock) Start() {

}

func (n ConstantLock) Stop() {

}
