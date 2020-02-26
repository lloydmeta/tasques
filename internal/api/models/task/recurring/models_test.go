package recurring

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/api/models/common"
	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
)

func retryTimesPtr(u task.RetryTimes) *task.RetryTimes {
	return &u
}

func priorityPtr(u task.Priority) *task.Priority {
	return &u
}

func durationPtr(u time.Duration) *Duration {
	v := Duration(u)
	return &v
}

func TestNewTask_ToDomainNewTask(t1 *testing.T) {
	type fields struct {
		ID                 recurring.Id
		ScheduleExpression recurring.ScheduleExpression
		TaskDefinition     TaskDefinition
	}
	type args struct {
		defaultRetryTimes        uint
		defaultProcessingTimeout time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   recurring.NewTask
	}{
		{
			name: "using of defaults for fields not passed",
			fields: fields{
				ID:                 "hello",
				ScheduleExpression: "* * * *",
				TaskDefinition: TaskDefinition{
					Queue: "q",
					Kind:  "k",
				},
			},
			args: args{
				defaultRetryTimes:        30,
				defaultProcessingTimeout: 1 * time.Hour,
			},
			want: recurring.NewTask{
				ID:                 "hello",
				ScheduleExpression: "* * * *",
				TaskDefinition: recurring.TaskDefinition{
					Queue:             "q",
					RetryTimes:        task.RetryTimes(30),
					Kind:              "k",
					Priority:          0,
					ProcessingTimeout: task.ProcessingTimeout(1 * time.Hour),
					Args:              nil,
					Context:           nil,
				},
			},
		},
		{
			name: "use of fields that are passed",
			fields: fields{
				ID:                 "hello2",
				ScheduleExpression: "0 * * * *",
				TaskDefinition: TaskDefinition{
					Queue:             "q2",
					RetryTimes:        retryTimesPtr(task.RetryTimes(3)),
					Kind:              "k2",
					Priority:          priorityPtr(task.Priority(1)),
					ProcessingTimeout: durationPtr(3 * time.Hour),
					Args:              &task.Args{"hi": "there"},
					Context:           &task.Context{"c": "tx"},
				},
			},
			args: args{
				defaultRetryTimes:        30,
				defaultProcessingTimeout: 1 * time.Hour,
			},
			want: recurring.NewTask{
				ID:                 "hello2",
				ScheduleExpression: "0 * * * *",
				TaskDefinition: recurring.TaskDefinition{
					Queue:             "q2",
					RetryTimes:        task.RetryTimes(3),
					Kind:              "k2",
					Priority:          1,
					ProcessingTimeout: task.ProcessingTimeout(3 * time.Hour),
					Args:              &task.Args{"hi": "there"},
					Context:           &task.Context{"c": "tx"},
				},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &NewTask{
				ID:                 tt.fields.ID,
				ScheduleExpression: tt.fields.ScheduleExpression,
				TaskDefinition:     tt.fields.TaskDefinition,
			}
			r := t.ToDomainNewTask(tt.args.defaultRetryTimes, tt.args.defaultProcessingTimeout)
			assert.EqualValues(t1, tt.want, r)
		})
	}
}

func TestFromDomainTask(t *testing.T) {
	now := time.Now().UTC()
	loadedAt := recurring.LoadedAt(now)
	type args struct {
		task *recurring.Task
	}
	tests := []struct {
		name string
		args args
		want Task
	}{
		{
			name: "basic test",
			args: args{
				task: &recurring.Task{
					ID:                 "test id",
					ScheduleExpression: "0 * * * *",
					TaskDefinition: recurring.TaskDefinition{
						Queue:             "q",
						RetryTimes:        0,
						Kind:              "k",
						Priority:          1,
						ProcessingTimeout: task.ProcessingTimeout(1 * time.Hour),
						Args: &task.Args{
							"hello": "world",
						},
						Context: &task.Context{
							"c": "ctx",
						},
					},
					IsDeleted: false,
					LoadedAt:  &loadedAt,
					Metadata: metadata.Metadata{
						CreatedAt:  metadata.CreatedAt(now),
						ModifiedAt: metadata.ModifiedAt(now),
						Version: metadata.Version{
							SeqNum:      123,
							PrimaryTerm: 456,
						},
					},
				},
			},
			want: Task{
				ID:                 "test id",
				ScheduleExpression: "0 * * * *",
				TaskDefinition: TaskDefinition{
					Queue:             "q",
					RetryTimes:        retryTimesPtr(task.RetryTimes(0)),
					Kind:              "k",
					Priority:          priorityPtr(task.Priority(1)),
					ProcessingTimeout: durationPtr(1 * time.Hour),
					Args: &task.Args{
						"hello": "world",
					},
					Context: &task.Context{
						"c": "ctx",
					},
				},
				LoadedAt: &now,
				Metadata: common.Metadata{
					CreatedAt:  now,
					ModifiedAt: now,
					Version: common.Version{
						SeqNum:      123,
						PrimaryTerm: 456,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromDomainTask(tt.args.task); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromDomainTask() = %v, want %v", got, tt.want)
			}
		})
	}
}
