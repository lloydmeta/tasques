package task

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/worker"
)

var now = time.Now().UTC()

func TestTask_RemainingRetries(t1 *testing.T) {

	type fields struct {
		ID             Id
		Queue          queue.Name
		RetryTimes     RetryTimes
		Retried        AttemptedTimes
		Kind           Kind
		State          State
		Priority       Priority
		RunAt          RunAt
		Args           *Args
		Context        *Context
		LastClaimed    *LastClaimed
		LastEnqueuedAt EnqueuedAt
		Metadata       metadata.Metadata
	}
	tests := []struct {
		name   string
		fields fields
		want   RemainingAttempts
	}{
		{
			name: "Zero retry times, zero retried",
			fields: fields{
				ID:             "test1",
				Queue:          "q",
				RetryTimes:     0,
				Retried:        0,
				Kind:           "something",
				State:          QUEUED,
				Priority:       1,
				RunAt:          RunAt(now),
				Args:           nil,
				Context:        nil,
				LastClaimed:    nil,
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			want: 1,
		},
		{
			name: "Zero retry times, non-zero retried (edge case)",
			fields: fields{
				ID:             "test1",
				Queue:          "q",
				RetryTimes:     0,
				Retried:        10,
				Kind:           "something",
				State:          QUEUED,
				Priority:       1,
				RunAt:          RunAt(now),
				Args:           nil,
				Context:        nil,
				LastClaimed:    nil,
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			want: 0,
		},
		{
			name: "Non-zero retry times, zero retried (edge case)",
			fields: fields{
				ID:             "test1",
				Queue:          "q",
				RetryTimes:     10,
				Retried:        0,
				Kind:           "something",
				State:          QUEUED,
				RunAt:          RunAt(now),
				Args:           nil,
				Context:        nil,
				LastClaimed:    nil,
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			want: 11,
		},
		{
			name: "Non-zero retry times, non-zero retried (edge case)",
			fields: fields{
				ID:             "test1",
				Queue:          "q",
				RetryTimes:     10,
				Retried:        5,
				Kind:           "something",
				State:          QUEUED,
				RunAt:          RunAt(now),
				Args:           nil,
				Context:        nil,
				LastClaimed:    nil,
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Task{
				ID:             tt.fields.ID,
				Queue:          tt.fields.Queue,
				RetryTimes:     tt.fields.RetryTimes,
				Attempted:      tt.fields.Retried,
				Kind:           tt.fields.Kind,
				State:          tt.fields.State,
				Priority:       tt.fields.Priority,
				RunAt:          tt.fields.RunAt,
				Args:           tt.fields.Args,
				Context:        tt.fields.Context,
				LastClaimed:    tt.fields.LastClaimed,
				LastEnqueuedAt: tt.fields.LastEnqueuedAt,
				Metadata:       tt.fields.Metadata,
			}
			assert.Equal(t1, tt.want, t.RemainingAttempts())
		})
	}
}

func TestTask_IntoClaimed(t1 *testing.T) {
	type fields struct {
		ID                Id
		Queue             queue.Name
		RetryTimes        RetryTimes
		Attempted         AttemptedTimes
		Kind              Kind
		State             State
		Priority          Priority
		RunAt             RunAt
		ProcessingTimeout ProcessingTimeout
		Args              *Args
		Context           *Context
		LastClaimed       *LastClaimed
		LastEnqueuedAt    EnqueuedAt
		Metadata          metadata.Metadata
	}
	type args struct {
		workerId          worker.Id
		at                ClaimedAt
		wantedLastClaimed LastClaimed
		wantedAttempted   AttemptedTimes
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Claiming a previously unclaimed task should work",
			fields: fields{
				ID:                "test1",
				Queue:             "q",
				RetryTimes:        10,
				Attempted:         5,
				Kind:              "something",
				State:             QUEUED,
				Priority:          3,
				RunAt:             RunAt(now),
				ProcessingTimeout: ProcessingTimeout(1 * time.Hour),
				Args:              nil,
				Context:           nil,
				LastClaimed:       nil,
				LastEnqueuedAt:    EnqueuedAt(now),
				Metadata:          metadata.Metadata{},
			},
			args: args{
				workerId: "werk",
				at:       ClaimedAt(now),
				wantedLastClaimed: LastClaimed{
					WorkerId:   "werk",
					ClaimedAt:  ClaimedAt(now),
					TimesOutAt: TimesOutAt(now.Add(1 * time.Hour)),
					Result:     nil,
				},
				wantedAttempted: 6,
			},
		},
		{
			name: "Claiming a previously claimed task should overwrite what was there before",
			fields: fields{
				ID:                "test1",
				Queue:             "q",
				RetryTimes:        10,
				Attempted:         5,
				Kind:              "something",
				State:             QUEUED,
				Priority:          3,
				RunAt:             RunAt(now),
				ProcessingTimeout: ProcessingTimeout(2 * time.Hour),
				Args:              nil,
				Context:           nil,
				LastClaimed: &LastClaimed{
					WorkerId:  "werk1",
					ClaimedAt: ClaimedAt(now),
					Result: &Result{
						Failure: &Failure{
							"something": "something",
						},
					},
					LastReport: &Report{
						At: ReportedAt(now),
						Data: &ReportedData{
							"old": "stuff",
						},
					},
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				workerId: "werk2",
				at:       ClaimedAt(now.Add(1 * time.Second)),
				wantedLastClaimed: LastClaimed{
					WorkerId:   "werk2",
					ClaimedAt:  ClaimedAt(now.Add(1 * time.Second)),
					TimesOutAt: TimesOutAt(now.Add(1*time.Second + 2*time.Hour)),
					Result:     nil,
					LastReport: nil,
				},
				wantedAttempted: 6,
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Task{
				ID:                tt.fields.ID,
				Queue:             tt.fields.Queue,
				RetryTimes:        tt.fields.RetryTimes,
				Attempted:         tt.fields.Attempted,
				Kind:              tt.fields.Kind,
				State:             tt.fields.State,
				Priority:          tt.fields.Priority,
				RunAt:             tt.fields.RunAt,
				ProcessingTimeout: tt.fields.ProcessingTimeout,
				Args:              tt.fields.Args,
				Context:           tt.fields.Context,
				LastClaimed:       tt.fields.LastClaimed,
				LastEnqueuedAt:    tt.fields.LastEnqueuedAt,
				Metadata:          tt.fields.Metadata,
			}
			t.IntoClaimed(tt.args.workerId, tt.args.at)
			assert.EqualValues(t1, &tt.args.wantedLastClaimed, t.LastClaimed)
			assert.EqualValues(t1, tt.args.wantedAttempted, t.Attempted)
		})
	}
}

func TestGenerateId(t *testing.T) {
	tests := []struct {
		name  string
		check func(id Id)
	}{
		{
			"should not have hyphens",
			func(id Id) {

				assert.NotContains(t, string(id), "-")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(GenerateId())
		})
	}
}

func TestTask_IntoDone(t1 *testing.T) {
	now := time.Now().UTC()
	type fields struct {
		ID             Id
		Queue          queue.Name
		RetryTimes     RetryTimes
		Retried        AttemptedTimes
		Kind           Kind
		State          State
		Priority       Priority
		RunAt          RunAt
		Args           *Args
		Context        *Context
		LastClaimed    *LastClaimed
		LastEnqueuedAt EnqueuedAt
		Metadata       metadata.Metadata
	}
	type args struct {
		byWorkerId worker.Id
		at         CompletedAt
		success    *Success
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErrType interface{}
	}{

		{
			name: "unclaimed task",
			fields: fields{
				ID:             "test1",
				Queue:          "q",
				RetryTimes:     10,
				Retried:        5,
				Kind:           "something",
				State:          QUEUED,
				Priority:       3,
				RunAt:          RunAt(now),
				Args:           nil,
				Context:        nil,
				LastClaimed:    nil,
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				at: CompletedAt(now),
				success: &Success{
					"hello": "there",
				},
			},
			wantErrType: NotClaimed{},
		},

		{
			name: "claimed task, owned by different worker",
			fields: fields{
				ID:         "test1",
				Queue:      "q",
				RetryTimes: 10,
				Retried:    5,
				Kind:       "something",
				State:      CLAIMED,
				Priority:   3,
				RunAt:      RunAt(now),
				Args:       nil,
				Context:    nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "work1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker2",
				at:         CompletedAt(now),
				success: &Success{
					"hello": "there",
				},
			},
			wantErrType: NotOwnedByWorker{},
		},

		{
			name: "claimed task",
			fields: fields{
				ID:         "test1",
				Queue:      "q",
				RetryTimes: 10,
				Retried:    5,
				Kind:       "something",
				State:      CLAIMED,
				Priority:   3,
				RunAt:      RunAt(now),
				Args:       nil,
				Context:    nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker1",
				at:         CompletedAt(now),
				success: &Success{
					"hello": "there",
				},
			},
			wantErrType: nil,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			task := &Task{
				ID:             tt.fields.ID,
				Queue:          tt.fields.Queue,
				RetryTimes:     tt.fields.RetryTimes,
				Attempted:      tt.fields.Retried,
				Kind:           tt.fields.Kind,
				State:          tt.fields.State,
				Priority:       tt.fields.Priority,
				RunAt:          tt.fields.RunAt,
				Args:           tt.fields.Args,
				Context:        tt.fields.Context,
				LastClaimed:    tt.fields.LastClaimed,
				LastEnqueuedAt: tt.fields.LastEnqueuedAt,
				Metadata:       tt.fields.Metadata,
			}
			err := task.IntoDone(tt.args.byWorkerId, tt.args.at, tt.args.success)
			if err != nil {
				if tt.wantErrType == nil {
					t1.Error(err)
				} else {
					assert.IsType(t1, tt.wantErrType, err)
				}
			} else {
				assert.EqualValues(t1, DONE, task.State)
				assert.EqualValues(t1, tt.args.at, task.LastClaimed.Result.At)
				assert.Nil(t1, task.LastClaimed.Result.Failure)
				assert.EqualValues(t1, tt.args.success, task.LastClaimed.Result.Success)
			}
		})
	}
}

func TestTask_IntoFailed(t1 *testing.T) {
	type fields struct {
		ID             Id
		Queue          queue.Name
		RetryTimes     RetryTimes
		Attempted      AttemptedTimes
		Kind           Kind
		State          State
		Priority       Priority
		RunAt          RunAt
		Args           *Args
		Context        *Context
		LastClaimed    *LastClaimed
		LastEnqueuedAt EnqueuedAt
		Metadata       metadata.Metadata
	}
	type args struct {
		byWorkerId worker.Id
		at         CompletedAt
		failure    *Failure
	}
	tests := []struct {
		name                    string
		fields                  fields
		args                    args
		wantedMaxNextRunAt      RunAt
		wantedRemainingAttempts RemainingAttempts
		wantState               State
		wantErrType             interface{}
	}{
		{
			name: "unclaimed task",
			fields: fields{
				ID:             "test1",
				Queue:          "q",
				RetryTimes:     10,
				Attempted:      5,
				Kind:           "something",
				State:          QUEUED,
				Priority:       3,
				RunAt:          RunAt(now),
				Args:           nil,
				Context:        nil,
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				at: CompletedAt(now),
				failure: &Failure{
					"hello": "there",
				},
			},
			wantErrType: NotClaimed{},
		},
		{
			name: "claimed task, owned by someone else",
			fields: fields{
				ID:         "test1",
				Queue:      "q",
				RetryTimes: 10,
				Attempted:  5,
				Kind:       "something",
				State:      CLAIMED,
				Priority:   3,
				RunAt:      RunAt(now),
				Args:       nil,
				Context:    nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker2",
				at:         CompletedAt(now),
				failure: &Failure{
					"hello": "there",
				},
			},
			wantErrType: NotOwnedByWorker{},
		},
		{
			name: "claimed task, retries available, first failure",
			fields: fields{
				ID:         "test1",
				Queue:      "q",
				RetryTimes: 10,
				Attempted:  1,
				Kind:       "something",
				State:      CLAIMED,
				Priority:   3,
				RunAt:      RunAt(now),
				Args:       nil,
				Context:    nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker1",
				at:         CompletedAt(now),
				failure: &Failure{
					"hello": "there",
				},
			},
			wantedMaxNextRunAt:      RunAt(now.Add((15 + 30*1) * time.Second)),
			wantedRemainingAttempts: 10,
			wantState:               FAILED,
			wantErrType:             nil,
		},
		{
			name: "claimed task, retries available",
			fields: fields{
				ID:         "test1",
				Queue:      "q",
				RetryTimes: 10,
				Attempted:  5,
				Kind:       "something",
				State:      CLAIMED,
				Priority:   3,
				RunAt:      RunAt(now),
				Args:       nil,
				Context:    nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker1",
				at:         CompletedAt(now),
				failure: &Failure{
					"hello": "there",
				},
			},
			wantedMaxNextRunAt:      RunAt(now.Add((15 + 4 ^ 4 + 30*5) * time.Second)),
			wantedRemainingAttempts: 6,
			wantState:               FAILED,
			wantErrType:             nil,
		},
		{
			name: "claimed task, no attempts available",
			fields: fields{
				ID:         "test1",
				Queue:      "q",
				RetryTimes: 10,
				Attempted:  11,
				Kind:       "something",
				State:      CLAIMED,
				Priority:   3,
				RunAt:      RunAt(now),
				Args:       nil,
				Context:    nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker1",
				at:         CompletedAt(now),
				failure: &Failure{
					"hello": "there",
				},
			},
			wantedMaxNextRunAt:      RunAt(now),
			wantedRemainingAttempts: 0,
			wantState:               DEAD,
			wantErrType:             nil,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			task := &Task{
				ID:             tt.fields.ID,
				Queue:          tt.fields.Queue,
				RetryTimes:     tt.fields.RetryTimes,
				Attempted:      tt.fields.Attempted,
				Kind:           tt.fields.Kind,
				State:          tt.fields.State,
				Priority:       tt.fields.Priority,
				RunAt:          tt.fields.RunAt,
				Args:           tt.fields.Args,
				Context:        tt.fields.Context,
				LastClaimed:    tt.fields.LastClaimed,
				LastEnqueuedAt: tt.fields.LastEnqueuedAt,
				Metadata:       tt.fields.Metadata,
			}
			err := task.IntoFailed(tt.args.byWorkerId, tt.args.at, tt.args.failure)
			if err != nil {
				if tt.wantErrType == nil {
					t1.Error(err)
				} else {
					assert.IsType(t1, tt.wantErrType, err)
				}
			} else {
				assert.EqualValues(t1, tt.wantedRemainingAttempts, task.RemainingAttempts())
				if task.State != DEAD {
					assert.Greater(t1, time.Time(task.RunAt).UnixNano(), time.Time(tt.args.at).UnixNano())
					assert.Less(t1, time.Time(task.RunAt).UnixNano(), time.Time(tt.wantedMaxNextRunAt).UnixNano())
				} else {
					// Doesn't change
					assert.Equal(t1, now.UnixNano(), time.Time(task.RunAt).UnixNano())
				}
				assert.EqualValues(t1, tt.wantState, task.State)
				assert.EqualValues(t1, tt.args.at, task.LastClaimed.Result.At)
				assert.Nil(t1, task.LastClaimed.Result.Success)
				assert.EqualValues(t1, tt.args.failure, task.LastClaimed.Result.Failure)
			}
		})
	}
}

func TestTask_IntoUnClaimed(t1 *testing.T) {
	type fields struct {
		ID             Id
		Queue          queue.Name
		RetryTimes     RetryTimes
		Retried        AttemptedTimes
		Kind           Kind
		State          State
		Priority       Priority
		RunAt          RunAt
		Args           *Args
		Context        *Context
		LastClaimed    *LastClaimed
		LastEnqueuedAt EnqueuedAt
		Metadata       metadata.Metadata
	}
	type args struct {
		byWorkerId worker.Id
		at         EnqueuedAt
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantState   State
		wantErrType interface{}
	}{
		{
			name: "unclaimed task",
			fields: fields{
				ID:             "test1",
				Queue:          "q",
				RetryTimes:     10,
				Retried:        5,
				Kind:           "something",
				State:          QUEUED,
				Priority:       3,
				RunAt:          RunAt(now),
				Args:           nil,
				Context:        nil,
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				at: EnqueuedAt(now),
			},
			wantErrType: NotClaimed{},
		},
		{
			name: "claimed task, owned by another worker",
			fields: fields{
				ID:         "test1",
				Queue:      "q",
				RetryTimes: 10,
				Retried:    5,
				Kind:       "something",
				State:      CLAIMED,
				Priority:   3,
				RunAt:      RunAt(now),
				Args:       nil,
				Context:    nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker2",
				at:         EnqueuedAt(now),
			},
			wantState:   QUEUED,
			wantErrType: NotOwnedByWorker{},
		},
		{
			name: "claimed task",
			fields: fields{
				ID:         "test1",
				Queue:      "q",
				RetryTimes: 10,
				Retried:    5,
				Kind:       "something",
				State:      CLAIMED,
				Priority:   3,
				RunAt:      RunAt(now),
				Args:       nil,
				Context:    nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker1",
				at:         EnqueuedAt(now),
			},
			wantState:   QUEUED,
			wantErrType: nil,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			task := &Task{
				ID:             tt.fields.ID,
				Queue:          tt.fields.Queue,
				RetryTimes:     tt.fields.RetryTimes,
				Attempted:      tt.fields.Retried,
				Kind:           tt.fields.Kind,
				State:          tt.fields.State,
				Priority:       tt.fields.Priority,
				RunAt:          tt.fields.RunAt,
				Args:           tt.fields.Args,
				Context:        tt.fields.Context,
				LastClaimed:    tt.fields.LastClaimed,
				LastEnqueuedAt: tt.fields.LastEnqueuedAt,
				Metadata:       tt.fields.Metadata,
			}
			err := task.IntoUnClaimed(tt.args.byWorkerId, tt.args.at)
			if err != nil {
				if tt.wantErrType == nil {
					t1.Error(err)
				} else {
					assert.IsType(t1, tt.wantErrType, err)
				}
			} else {
				assert.EqualValues(t1, tt.wantState, task.State)
				assert.EqualValues(t1, tt.args.at, task.LastEnqueuedAt)
			}
		})
	}
}

func TestTask_ReportIn(t1 *testing.T) {
	report := NewReport{Data: &ReportedData{"message": "touching base"}}
	type fields struct {
		ID                Id
		Queue             queue.Name
		RetryTimes        RetryTimes
		Retried           AttemptedTimes
		Kind              Kind
		State             State
		Priority          Priority
		RunAt             RunAt
		ProcessingTimeout ProcessingTimeout
		Args              *Args
		Context           *Context
		LastClaimed       *LastClaimed
		LastEnqueuedAt    EnqueuedAt
		Metadata          metadata.Metadata
	}
	type args struct {
		byWorkerId worker.Id
		at         ReportedAt
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErrType interface{}
	}{
		{
			name: "unclaimed task",
			fields: fields{
				ID:                "test1",
				Queue:             "q",
				RetryTimes:        10,
				Retried:           5,
				Kind:              "something",
				State:             QUEUED,
				Priority:          3,
				ProcessingTimeout: ProcessingTimeout(1 * time.Hour),
				RunAt:             RunAt(now),
				Args:              nil,
				Context:           nil,
				LastEnqueuedAt:    EnqueuedAt(now),
				Metadata:          metadata.Metadata{},
			},
			args: args{
				at: ReportedAt(now),
			},
			wantErrType: NotClaimed{},
		},
		{
			name: "claimed task, owned by another worker",
			fields: fields{
				ID:                "test1",
				Queue:             "q",
				RetryTimes:        10,
				Retried:           5,
				Kind:              "something",
				State:             CLAIMED,
				Priority:          3,
				RunAt:             RunAt(now),
				ProcessingTimeout: ProcessingTimeout(1 * time.Hour),
				Args:              nil,
				Context:           nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker2",
				at:         ReportedAt(now),
			},
			wantErrType: NotOwnedByWorker{},
		},
		{
			name: "claimed task, but filing report from the past",
			fields: fields{
				ID:                "test1",
				Queue:             "q",
				RetryTimes:        10,
				Retried:           5,
				Kind:              "something",
				State:             CLAIMED,
				Priority:          3,
				RunAt:             RunAt(now),
				ProcessingTimeout: ProcessingTimeout(1 * time.Hour),
				Args:              nil,
				Context:           nil,
				LastClaimed: &LastClaimed{
					WorkerId:  "worker1",
					ClaimedAt: ClaimedAt{},
					LastReport: &Report{
						At:   ReportedAt(now),
						Data: nil,
					},
					Result: nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker1",
				at:         ReportedAt(now.Add(1 * time.Hour)),
			},
			wantErrType: ReportFromThePast{},
		},
		{
			name: "claimed task",
			fields: fields{
				ID:                "test1",
				Queue:             "q",
				RetryTimes:        10,
				Retried:           5,
				Kind:              "something",
				State:             CLAIMED,
				Priority:          3,
				RunAt:             RunAt(now),
				ProcessingTimeout: ProcessingTimeout(1 * time.Hour),
				Args:              nil,
				Context:           nil,
				LastClaimed: &LastClaimed{
					WorkerId:   "worker1",
					ClaimedAt:  ClaimedAt{},
					LastReport: nil,
					Result:     nil,
				},
				LastEnqueuedAt: EnqueuedAt(now),
				Metadata:       metadata.Metadata{},
			},
			args: args{
				byWorkerId: "worker1",
				at:         ReportedAt(now),
			},
			wantErrType: nil,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			task := &Task{
				ID:                tt.fields.ID,
				Queue:             tt.fields.Queue,
				RetryTimes:        tt.fields.RetryTimes,
				Attempted:         tt.fields.Retried,
				Kind:              tt.fields.Kind,
				State:             tt.fields.State,
				Priority:          tt.fields.Priority,
				RunAt:             tt.fields.RunAt,
				ProcessingTimeout: tt.fields.ProcessingTimeout,
				Args:              tt.fields.Args,
				Context:           tt.fields.Context,
				LastClaimed:       tt.fields.LastClaimed,
				LastEnqueuedAt:    tt.fields.LastEnqueuedAt,
				Metadata:          tt.fields.Metadata,
			}
			err := task.ReportIn(tt.args.byWorkerId, report, tt.args.at)
			if err != nil {
				if tt.wantErrType == nil {
					t1.Error(err)
				} else {
					assert.IsType(t1, tt.wantErrType, err)
				}
			} else {
				assert.EqualValues(t1, time.Time(tt.args.at).Add(1*time.Hour), task.LastClaimed.TimesOutAt)
				assert.EqualValues(t1, tt.args.at, task.LastClaimed.LastReport.At)
				assert.EqualValues(t1, tt.args.at, task.LastClaimed.LastReport.At)
				assert.EqualValues(t1, report.Data, task.LastClaimed.LastReport.Data)
			}
		})
	}
}

func Test_State_JSON_serialising(t1 *testing.T) {
	type fields struct {
		State State
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			"queued",
			fields{
				State: QUEUED,
			},
			[]byte("\"queued\""),
		},
		{
			"claimed",
			fields{
				State: CLAIMED,
			},
			[]byte("\"claimed\""),
		},
		{
			"failed",
			fields{
				State: FAILED,
			},
			[]byte("\"failed\""),
		},
		{
			"done",
			fields{
				State: DONE,
			},
			[]byte("\"done\""),
		},
		{
			"dead",
			fields{
				State: DEAD,
			},
			[]byte("\"dead\""),
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			serialised, err := json.Marshal(tt.fields.State)
			assert.NoError(t1, err)
			assert.EqualValues(t1, tt.want, serialised)
		})
	}
}

func Test_State_JSON_deserialising(t1 *testing.T) {
	type fields struct {
		bytes []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   State
	}{
		{
			"queued",
			fields{
				bytes: []byte("\"queued\""),
			},
			QUEUED,
		},
		{
			"claimed",
			fields{
				bytes: []byte("\"claimed\""),
			},
			CLAIMED,
		},
		{
			"failed",
			fields{
				bytes: []byte("\"failed\""),
			},
			FAILED,
		},
		{
			"done",
			fields{
				bytes: []byte("\"done\""),
			},
			DONE,
		},
		{
			"dead",
			fields{
				bytes: []byte("\"dead\""),
			},
			DEAD,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			var s State
			err := json.Unmarshal(tt.fields.bytes, &s)
			assert.NoError(t1, err)
			assert.EqualValues(t1, tt.want, s)
		})
	}
}
