package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/api/models/task"
	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	domainTask "github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/worker"
)

func TestNewTasksController(t *testing.T) {
	type args struct {
		tasksService domainTask.Service
		config       config.TasksDefaults
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"should not panic",
			args{
				tasksService: &domainTask.MockTasksService{},
				config:       tasksConfig,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { New(tt.args.tasksService, tasksConfig) })
		})
	}
}

func Test_handleErr(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name     string
		args     args
		wantCode int
	}{
		{
			"random errors should 500",
			args{
				fmt.Errorf("wtf"),
			},
			500,
		},
		{
			"InvalidPersistedData errors should 500",
			args{
				domainTask.InvalidPersistedData{},
			},
			500,
		},
		{
			"NotFound errors should 404",
			args{
				domainTask.NotFound{},
			},
			404,
		},
		{
			"NotClaimed errors should 400",
			args{
				domainTask.NotClaimed{},
			},
			400,
		},
		{
			"ReportFromThePast errors should 400",
			args{
				domainTask.ReportFromThePast{},
			},
			400,
		},
		{
			"NotOwnedByWorker errors should 400",
			args{
				domainTask.NotOwnedByWorker{},
			},
			403,
		},
		{
			"InvalidVersion errors should 409",
			args{
				domainTask.InvalidVersion{},
			},
			409,
		},
		{
			"AlreadyExists errors should 409",
			args{
				domainTask.AlreadyExists{},
			},
			409,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleErr(tt.args.err)
			assert.EqualValues(t, tt.wantCode, got.StatusCode)
		})
	}
}

func Test_tasksControllerImpl_Claim(t *testing.T) {
	type fields struct {
		tasksService *domainTask.MockTasksService
	}
	type args struct {
		workerId worker.Id
		queues   []queue.Name
		number   uint
		blockFor time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []task.Task
		wantErr bool
	}{
		{
			"successful service return",
			fields{tasksService: &domainTask.MockTasksService{}},
			args{
				workerId: "werk",
				queues:   []queue.Name{"q"},
				number:   10,
				blockFor: 1 * time.Second,
			},
			[]task.Task{mockApiTask},
			false,
		},
		{
			"failed service return",
			fields{tasksService: &domainTask.MockTasksService{
				ClaimOverride: func() (tasks []domainTask.Task, err error) {
					return nil, fmt.Errorf("boom")
				},
			}},
			args{
				workerId: "werk",
				queues:   []queue.Name{"q"},
				number:   10,
				blockFor: 1 * time.Second,
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				tasksService: tt.fields.tasksService,
				tasksConfig:  tasksConfig,
				getNowUtc:    func() time.Time { return time.Now().UTC() },
			}
			got, err := c.Claim(context.Background(), tt.args.workerId, tt.args.queues, tt.args.number, tt.args.blockFor)
			assert.EqualValues(t, 1, tt.fields.tasksService.ClaimCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_tasksControllerImpl_Create(t *testing.T) {
	type fields struct {
		tasksService *domainTask.MockTasksService
	}
	type args struct {
		newTask *task.NewTask
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *task.Task
		wantErr bool
	}{
		{
			"successful service return",
			fields{tasksService: &domainTask.MockTasksService{}},
			args{
				&task.NewTask{
					Queue: "q",
					Kind:  "go",
				},
			},
			&mockApiTask,
			false,
		},
		{
			"failed service return",
			fields{tasksService: &domainTask.MockTasksService{
				CreateOverride: func() (tasks *domainTask.Task, err error) {
					return nil, fmt.Errorf("yikes")
				},
			}},
			args{
				&task.NewTask{
					Queue: "q",

					Kind: "go",
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				tasksService: tt.fields.tasksService,
				tasksConfig:  tasksConfig,
				getNowUtc:    func() time.Time { return time.Now().UTC() },
			}
			got, err := c.Create(context.Background(), tt.args.newTask)
			assert.EqualValues(t, 1, tt.fields.tasksService.CreateCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_tasksControllerImpl_Get(t *testing.T) {
	type fields struct {
		tasksService *domainTask.MockTasksService
	}
	type args struct {
		queue  queue.Name
		taskId domainTask.Id
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *task.Task
		wantErr bool
	}{
		{
			"successful service return",
			fields{tasksService: &domainTask.MockTasksService{}},
			args{
				"q",
				"id",
			},
			&mockApiTask,
			false,
		},
		{
			"failed service return",
			fields{tasksService: &domainTask.MockTasksService{
				GetOverride: func() (d *domainTask.Task, err error) {
					return nil, fmt.Errorf("yikes")
				},
			}},
			args{
				"q",
				"id",
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				tasksService: tt.fields.tasksService,
				tasksConfig:  tasksConfig,
				getNowUtc:    func() time.Time { return time.Now().UTC() },
			}
			got, err := c.Get(context.Background(), tt.args.queue, tt.args.taskId)
			assert.EqualValues(t, 1, tt.fields.tasksService.GetCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_tasksControllerImpl_MarkDone(t *testing.T) {
	type fields struct {
		tasksService *domainTask.MockTasksService
	}
	type args struct {
		workerId worker.Id
		queue    queue.Name
		taskId   domainTask.Id
		success  *domainTask.Success
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *task.Task
		wantErr bool
	}{
		{
			"successful service return",
			fields{tasksService: &domainTask.MockTasksService{}},
			args{
				workerId: "worker",
				queue:    "q",
				taskId:   "taskId",
			},
			&mockApiTask,
			false,
		},
		{
			"failed service return",
			fields{tasksService: &domainTask.MockTasksService{
				MarkDoneOverride: func() (d *domainTask.Task, err error) {
					return nil, fmt.Errorf("nope")
				},
			}},
			args{
				workerId: "worker",
				queue:    "q",
				taskId:   "taskId",
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				tasksService: tt.fields.tasksService,
				tasksConfig:  tasksConfig,
				getNowUtc:    func() time.Time { return time.Now().UTC() },
			}
			got, err := c.MarkDone(context.Background(), tt.args.workerId, tt.args.queue, tt.args.taskId, tt.args.success)
			assert.EqualValues(t, 1, tt.fields.tasksService.MarkDoneCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_tasksControllerImpl_MarkFailed(t *testing.T) {
	type fields struct {
		tasksService *domainTask.MockTasksService
	}
	type args struct {
		workerId worker.Id
		queue    queue.Name
		taskId   domainTask.Id
		failure  *domainTask.Failure
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *task.Task
		wantErr bool
	}{
		{
			"successful service return",
			fields{tasksService: &domainTask.MockTasksService{}},
			args{
				workerId: "worker",
				queue:    "q",
				taskId:   "taskId",
			},
			&mockApiTask,
			false,
		},
		{
			"failed service return",
			fields{tasksService: &domainTask.MockTasksService{
				MarkFailedOverride: func() (d *domainTask.Task, err error) {
					return nil, fmt.Errorf("nope")
				},
			}},
			args{
				workerId: "worker",
				queue:    "q",
				taskId:   "taskId",
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				tasksService: tt.fields.tasksService,
				tasksConfig:  tasksConfig,
				getNowUtc:    func() time.Time { return time.Now().UTC() },
			}
			got, err := c.MarkFailed(context.Background(), tt.args.workerId, tt.args.queue, tt.args.taskId, tt.args.failure)
			assert.EqualValues(t, 1, tt.fields.tasksService.MarkFailedCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_tasksControllerImpl_ReportIn(t *testing.T) {
	type fields struct {
		tasksService *domainTask.MockTasksService
	}
	type args struct {
		workerId worker.Id
		queue    queue.Name
		taskId   domainTask.Id
		report   task.NewReport
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *task.Task
		wantErr bool
	}{
		{
			"successful service return",
			fields{tasksService: &domainTask.MockTasksService{}},
			args{
				workerId: "worker",
				queue:    "q",
				taskId:   "taskId",
				report: task.NewReport{
					Data: nil,
				},
			},
			&mockApiTask,
			false,
		},
		{
			"failed service return",
			fields{tasksService: &domainTask.MockTasksService{
				ReportInOverride: func() (d *domainTask.Task, err error) {
					return nil, fmt.Errorf("nope")
				},
			}},
			args{
				workerId: "worker",
				queue:    "q",
				taskId:   "taskId",
				report: task.NewReport{
					Data: nil,
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				tasksService: tt.fields.tasksService,
				tasksConfig:  tasksConfig,
				getNowUtc:    func() time.Time { return time.Now().UTC() },
			}
			got, err := c.ReportIn(context.Background(), tt.args.workerId, tt.args.queue, tt.args.taskId, tt.args.report)
			assert.EqualValues(t, 1, tt.fields.tasksService.ReportInCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_tasksControllerImpl_UnClaim(t *testing.T) {
	type fields struct {
		tasksService *domainTask.MockTasksService
	}
	type args struct {
		workerId worker.Id
		queue    queue.Name
		taskId   domainTask.Id
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *task.Task
		wantErr bool
	}{
		{
			"successful service return",
			fields{tasksService: &domainTask.MockTasksService{}},
			args{
				workerId: "worker",
				queue:    "q",
				taskId:   "taskId",
			},
			&mockApiTask,
			false,
		},
		{
			"failed service return",
			fields{tasksService: &domainTask.MockTasksService{
				UnClaimOverride: func() (d *domainTask.Task, err error) {
					return nil, fmt.Errorf("nope")
				},
			}},
			args{
				workerId: "worker",
				queue:    "q",
				taskId:   "taskId",
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				tasksService: tt.fields.tasksService,
				tasksConfig:  tasksConfig,
				getNowUtc:    func() time.Time { return time.Now().UTC() },
			}
			got, err := c.UnClaim(context.Background(), tt.args.workerId, tt.args.queue, tt.args.taskId)
			assert.EqualValues(t, 1, tt.fields.tasksService.UnClaimCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

var tasksConfig = config.TasksDefaults{
	BlockFor:                    100 * time.Millisecond,
	BlockForRetryMinWait:        25 * time.Millisecond,
	BlockForRetryMaxRetries:     100,
	WorkerProcessingTimeout:     30 * time.Minute,
	ClaimAmount:                 5,
	ClaimAmountSearchMultiplier: 10,
	RetryTimes:                  10,
	VersionConflictRetryTimes:   50,
}

var mockApiTask = task.FromDomainTask(&domainTask.MockDomainTask)
