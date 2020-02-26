package task

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/api/models/common"
	"github.com/lloydmeta/tasques/internal/domain/queue"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/task"
)

func retryTimesPtr(u task.RetryTimes) *task.RetryTimes {
	return &u
}

func priorityPtr(u task.Priority) *task.Priority {
	return &u
}

func timePtr(u time.Time) *time.Time {
	return &u
}

func durationPtr(u time.Duration) *time.Duration {
	return &u
}

func TestNewTask_ToDomainNewTask(t1 *testing.T) {
	now := time.Now().UTC()
	type fields struct {
		Queue             queue.Name
		RetryTimes        *task.RetryTimes
		Kind              task.Kind
		Priority          *task.Priority
		ProcessingTimeout *Duration
		RunAt             *time.Time
		Args              *task.Args
		Context           *task.Context
	}
	type args struct {
		defaultRetryTimes    uint
		defaultRunAt         time.Time
		defaultClaimsTimeout time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   task.NewTask
	}{
		{
			"should use expected defaults for things not passed",
			fields{
				Queue: queue.Name("q"),
				Kind:  "run",
			},
			args{
				defaultRetryTimes:    10,
				defaultRunAt:         now,
				defaultClaimsTimeout: 5 * time.Second,
			},
			task.NewTask{
				Queue:             "q",
				RetryTimes:        task.RetryTimes(10),
				Kind:              "run",
				Priority:          0,
				ProcessingTimeout: task.ProcessingTimeout(5 * time.Second),
				RunAt:             task.RunAt(now),
				Args:              nil,
				Context:           nil,
			},
		},
		{
			"should use expected defaults runAt if 0",
			fields{
				Queue: queue.Name("q"),
				Kind:  "run",
				RunAt: &TimeZero,
			},
			args{
				defaultRetryTimes:    10,
				defaultRunAt:         now,
				defaultClaimsTimeout: 5 * time.Second,
			},
			task.NewTask{
				Queue:             "q",
				RetryTimes:        task.RetryTimes(10),
				Kind:              "run",
				Priority:          0,
				ProcessingTimeout: task.ProcessingTimeout(5 * time.Second),
				RunAt:             task.RunAt(now),
				Args:              nil,
				Context:           nil,
			},
		},
		{
			"should use use the full spectrum of things I pass",
			fields{
				Queue:             queue.Name("q"),
				RetryTimes:        retryTimesPtr(999),
				Kind:              "run",
				Priority:          priorityPtr(100),
				ProcessingTimeout: (*Duration)(durationPtr(1 * time.Hour)),
				RunAt:             timePtr(now.Add(1 * time.Hour)),
				Args: &task.Args{
					"arr": "matey",
				},
				Context: &task.Context{
					"society": "man",
				},
			},
			args{
				defaultRetryTimes:    10,
				defaultRunAt:         now,
				defaultClaimsTimeout: 5 * time.Second,
			},
			task.NewTask{
				Queue:             "q",
				RetryTimes:        task.RetryTimes(999),
				Kind:              "run",
				Priority:          task.Priority(100),
				ProcessingTimeout: task.ProcessingTimeout(1 * time.Hour),
				RunAt:             task.RunAt(now.Add(1 * time.Hour)),
				Args: &task.Args{
					"arr": "matey",
				},
				Context: &task.Context{
					"society": "man",
				},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &NewTask{
				Queue:             tt.fields.Queue,
				RetryTimes:        tt.fields.RetryTimes,
				Kind:              tt.fields.Kind,
				Priority:          tt.fields.Priority,
				ProcessingTimeout: tt.fields.ProcessingTimeout,
				RunAt:             tt.fields.RunAt,
				Args:              tt.fields.Args,
				Context:           tt.fields.Context,
			}
			got := t.ToDomainNewTask(tt.args.defaultRetryTimes, tt.args.defaultRunAt, tt.args.defaultClaimsTimeout)
			assert.EqualValues(t1, tt.want, got)
		})
	}
}

func TestFromDomainTask(t *testing.T) {
	now := time.Now().UTC()
	type args struct {
		dTask *task.Task
	}
	tests := []struct {
		name string
		args args
		want Task
	}{
		{
			"when there are empty attributes",
			args{
				&task.Task{
					ID:                "id",
					Queue:             "q",
					RetryTimes:        2,
					Attempted:         1,
					Kind:              "run",
					State:             task.CLAIMED,
					ProcessingTimeout: task.ProcessingTimeout(1 * time.Hour),
					Priority:          15,
					RunAt:             task.RunAt(now),
					Args:              nil,
					Context:           nil,
					LastClaimed:       nil,
					LastEnqueuedAt:    task.EnqueuedAt(now.Add(1 * time.Second)),
					Metadata: metadata.Metadata{
						CreatedAt:  metadata.CreatedAt(now.Add(2 * time.Second)),
						ModifiedAt: metadata.ModifiedAt(now.Add(3 * time.Second)),
						Version: metadata.Version{
							SeqNum:      1,
							PrimaryTerm: 2,
						},
					},
				},
			},
			Task{
				ID:                "id",
				Queue:             "q",
				RetryTimes:        2,
				Attempted:         1,
				Kind:              "run",
				State:             task.CLAIMED,
				Priority:          15,
				RunAt:             now,
				Args:              nil,
				ProcessingTimeout: Duration(1 * time.Hour),
				Context:           nil,
				LastClaimed:       nil,
				LastEnqueuedAt:    now.Add(1 * time.Second),
				Metadata: common.Metadata{
					CreatedAt:  now.Add(2 * time.Second),
					ModifiedAt: now.Add(3 * time.Second),
					Version: common.Version{
						SeqNum:      1,
						PrimaryTerm: 2,
					},
				},
			},
		},
		{
			"when there are filled attributes (failed task)",
			args{
				&task.Task{
					ID:                "id",
					Queue:             "q",
					RetryTimes:        2,
					Attempted:         1,
					Kind:              "run",
					State:             task.CLAIMED,
					Priority:          15,
					RunAt:             task.RunAt(now),
					ProcessingTimeout: task.ProcessingTimeout(2 * time.Hour),
					Args: &task.Args{
						"arr": "matey",
					},
					Context: &task.Context{
						"reqId": "asdf",
					},
					LastClaimed: &task.LastClaimed{
						WorkerId:  "werk",
						ClaimedAt: task.ClaimedAt(now.Add(4 * time.Second)),
						LastReport: &task.Report{
							At: task.ReportedAt(now.Add(5 * time.Second)),
							Data: &task.ReportedData{
								"something": "else",
							},
						},
						Result: &task.Result{
							At: task.CompletedAt(now.Add(6 * time.Second)),
							Failure: &task.Failure{
								"oops": "nooo!",
							},
							Success: nil,
						},
					},
					LastEnqueuedAt: task.EnqueuedAt(now.Add(1 * time.Second)),
					Metadata: metadata.Metadata{
						CreatedAt:  metadata.CreatedAt(now.Add(2 * time.Second)),
						ModifiedAt: metadata.ModifiedAt(now.Add(3 * time.Second)),
						Version: metadata.Version{
							SeqNum:      1,
							PrimaryTerm: 2,
						},
					},
				},
			},
			Task{
				ID:                "id",
				Queue:             "q",
				RetryTimes:        2,
				Attempted:         1,
				Kind:              "run",
				State:             task.CLAIMED,
				Priority:          15,
				RunAt:             now,
				ProcessingTimeout: Duration(2 * time.Hour),
				Args: &task.Args{
					"arr": "matey",
				},
				Context: &task.Context{
					"reqId": "asdf",
				},
				LastClaimed: &LastClaimed{
					WorkerId:  "werk",
					ClaimedAt: now.Add(4 * time.Second),
					LastReport: &Report{
						At: now.Add(5 * time.Second),
						Data: &task.ReportedData{
							"something": "else",
						},
					},
					Result: &Result{
						At: now.Add(6 * time.Second),
						Failure: &task.Failure{
							"oops": "nooo!",
						},
						Success: nil,
					},
				},
				LastEnqueuedAt: now.Add(1 * time.Second),
				Metadata: common.Metadata{
					CreatedAt:  now.Add(2 * time.Second),
					ModifiedAt: now.Add(3 * time.Second),
					Version: common.Version{
						SeqNum:      1,
						PrimaryTerm: 2,
					},
				},
			},
		}, {
			"when there are filled attributes (successful task)",
			args{
				&task.Task{
					ID:                "id",
					Queue:             "q",
					RetryTimes:        2,
					Attempted:         1,
					Kind:              "run",
					State:             task.CLAIMED,
					Priority:          15,
					RunAt:             task.RunAt(now),
					ProcessingTimeout: task.ProcessingTimeout(2 * time.Hour),
					Args: &task.Args{
						"arr": "matey",
					},
					Context: &task.Context{
						"reqId": "asdf",
					},
					LastClaimed: &task.LastClaimed{
						WorkerId:  "werk",
						ClaimedAt: task.ClaimedAt(now.Add(4 * time.Second)),
						LastReport: &task.Report{
							At: task.ReportedAt(now.Add(5 * time.Second)),
							Data: &task.ReportedData{
								"something": "else",
							},
						},
						Result: &task.Result{
							At:      task.CompletedAt(now.Add(6 * time.Second)),
							Failure: nil,
							Success: &task.Success{
								"ok": "yay",
							},
						},
					},
					LastEnqueuedAt: task.EnqueuedAt(now.Add(1 * time.Second)),
					Metadata: metadata.Metadata{
						CreatedAt:  metadata.CreatedAt(now.Add(2 * time.Second)),
						ModifiedAt: metadata.ModifiedAt(now.Add(3 * time.Second)),
						Version: metadata.Version{
							SeqNum:      1,
							PrimaryTerm: 2,
						},
					},
				},
			},
			Task{
				ID:                "id",
				Queue:             "q",
				RetryTimes:        2,
				Attempted:         1,
				Kind:              "run",
				State:             task.CLAIMED,
				Priority:          15,
				RunAt:             now,
				ProcessingTimeout: Duration(2 * time.Hour),
				Args: &task.Args{
					"arr": "matey",
				},
				Context: &task.Context{
					"reqId": "asdf",
				},
				LastClaimed: &LastClaimed{
					WorkerId:  "werk",
					ClaimedAt: now.Add(4 * time.Second),
					LastReport: &Report{
						At: now.Add(5 * time.Second),
						Data: &task.ReportedData{
							"something": "else",
						},
					},
					Result: &Result{
						At:      now.Add(6 * time.Second),
						Failure: nil,
						Success: &task.Success{
							"ok": "yay",
						},
					},
				},
				LastEnqueuedAt: now.Add(1 * time.Second),
				Metadata: common.Metadata{
					CreatedAt:  now.Add(2 * time.Second),
					ModifiedAt: now.Add(3 * time.Second),
					Version: common.Version{
						SeqNum:      1,
						PrimaryTerm: 2,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromDomainTask(tt.args.dTask)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
