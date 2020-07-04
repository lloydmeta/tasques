package recurring

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
	"github.com/lloydmeta/tasques/internal/infra/apm/tracing"
)

func Test_NewScheduler(t *testing.T) {
	assert.NotPanics(t, func() {
		NewScheduler(&task.MockTasksService{}, tracing.NoopTracer{})
	})
}

func Test_schedulerImpl_Schedule(t *testing.T) {
	expectedI := 10
	is := make(chan int)
	taskToSchedule := recurring.Task{
		ID:                 "send-me-an-int",
		ScheduleExpression: "@every 1s",
		TaskDefinition:     recurring.TaskDefinition{},
		IsDeleted:          false,
		LoadedAt:           nil,
		Metadata:           metadata.Metadata{},
	}
	tasksService := task.MockTasksService{
		CreateOverride: func() (t *task.Task, err error) {
			is <- expectedI
			return &task.MockDomainTask, nil
		},
	}
	scheduler := &schedulerImpl{
		cron:          cron.New(cron.WithLocation(time.UTC)),
		tasksService:  &tasksService,
		tracer:        tracing.NoopTracer{},
		idsToEntryIds: make(map[task.RecurringTaskId]cron.EntryID),
		mu:            sync.Mutex{},
		getUTC: func() time.Time {
			return time.Now().UTC()
		},
	}
	scheduler.Start()
	err := scheduler.Schedule(taskToSchedule)
	if err != nil {
		t.Error(err)
	}
	select {
	case <-time.NewTicker(5 * time.Second).C:
		assert.Fail(t, "didn't get a message")
	case received := <-is:
		assert.Equal(t, expectedI, received)
		_, taskIdPresent := scheduler.idsToEntryIds[taskToSchedule.ID]
		assert.True(t, taskIdPresent)
	}
	scheduler.Stop()
	assert.EqualValues(t, 0, tasksService.RefreshAsNeededCalled)
	assert.EqualValues(t, 0, tasksService.OutstandingTasksCountCalled)
}

func Test_schedulerImpl_Schedule_skipIfOutstandingTasksExist_tasks_exist(t *testing.T) {
	tests := []struct {
		name                          string
		outstandingTasksCountOverride func() (u uint, err error)
		shouldCallTaskCreate          bool
	}{
		{
			name: "if there are outstanding tasks",
			outstandingTasksCountOverride: func() (u uint, err error) {
				return 1, nil
			},
			shouldCallTaskCreate: false,
		},
		{
			name: "if there are no outstanding tasks",
			outstandingTasksCountOverride: func() (u uint, err error) {
				return 0, nil
			},
			shouldCallTaskCreate: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasksService := task.MockTasksService{
				CreateOverride: func() (t *task.Task, err error) {
					return &task.MockDomainTask, nil
				},
				OutstandingTasksCountOverride: tt.outstandingTasksCountOverride,
			}
			scheduler := &schedulerImpl{
				cron:          cron.New(cron.WithLocation(time.UTC)),
				tasksService:  &tasksService,
				tracer:        tracing.NoopTracer{},
				idsToEntryIds: make(map[task.RecurringTaskId]cron.EntryID),
				mu:            sync.Mutex{},
				getUTC: func() time.Time {
					return time.Now().UTC()
				},
			}
			scheduler.Start()
			err := scheduler.Schedule(skipIfOutstandingTaskToSchedule)
			if err != nil {
				t.Error(err)
			}
			time.Sleep(5 * time.Second)
			scheduler.Stop()
			assert.GreaterOrEqual(t, tasksService.RefreshAsNeededCalled, uint(1))
			assert.GreaterOrEqual(t, tasksService.OutstandingTasksCountCalled, uint(1))
			if tt.shouldCallTaskCreate {
				assert.GreaterOrEqual(t, tasksService.CreateCalled, uint(1))
			} else {
				assert.EqualValues(t, 0, tasksService.CreateCalled)
			}
			// It should nonetheless be scheduled
			_, taskIdPresent := scheduler.idsToEntryIds[skipIfOutstandingTaskToSchedule.ID]
			assert.True(t, taskIdPresent)
		})

	}
}
func Test_schedulerImpl_Schedule_skipIfOutstandingTasksExist_error_handling(t *testing.T) {
	tests := []struct {
		name                          string
		refreshAsNeededOverride       func() error
		outstandingTasksCountOverride func() (u uint, err error)
		shouldCallTasksCount          bool
		shouldCallTaskCreate          bool
	}{
		{
			name: "if refresh fails",
			refreshAsNeededOverride: func() error {
				return fmt.Errorf("dangit")
			},
			shouldCallTasksCount: true,
			shouldCallTaskCreate: true,
		},
		{
			name: "if count outstanding fails",
			outstandingTasksCountOverride: func() (u uint, err error) {
				return 0, fmt.Errorf("dangit")
			},
			shouldCallTasksCount: true,
			shouldCallTaskCreate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasksService := task.MockTasksService{
				CreateOverride: func() (t *task.Task, err error) {
					return nil, fmt.Errorf("dang")
				},
				RefreshAsNeededOverride:       tt.refreshAsNeededOverride,
				OutstandingTasksCountOverride: tt.outstandingTasksCountOverride,
			}
			scheduler := &schedulerImpl{
				cron:          cron.New(cron.WithLocation(time.UTC)),
				tasksService:  &tasksService,
				tracer:        tracing.NoopTracer{},
				idsToEntryIds: make(map[task.RecurringTaskId]cron.EntryID),
				mu:            sync.Mutex{},
				getUTC: func() time.Time {
					return time.Now().UTC()
				},
			}
			scheduler.Start()
			assert.NotPanics(t, func() {
				err := scheduler.Schedule(skipIfOutstandingTaskToSchedule)
				if err != nil {
					t.Error(err)
				}
				time.Sleep(5 * time.Second)
				scheduler.Stop()
				if tt.shouldCallTasksCount {
					assert.GreaterOrEqual(t, tasksService.OutstandingTasksCountCalled, uint(1))
				} else {
					assert.EqualValues(t, 0, tasksService.OutstandingTasksCountCalled)
				}
				if tt.shouldCallTaskCreate {
					assert.GreaterOrEqual(t, tasksService.CreateCalled, uint(1))
				} else {
					assert.EqualValues(t, 0, tasksService.CreateCalled)
				}
				// It should nonetheless be scheduled
				_, taskIdPresent := scheduler.idsToEntryIds[skipIfOutstandingTaskToSchedule.ID]
				assert.True(t, taskIdPresent)
			})
		})

	}
}

func Test_schedulerImpl_Unschedule(t *testing.T) {
	type fields struct {
		tasksService  task.Service
		idsToEntryIds map[task.RecurringTaskId]cron.EntryID
	}
	type args struct {
		taskId task.RecurringTaskId
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "unscheduling something that isn't scheduled",
			fields: fields{
				tasksService:  &task.MockTasksService{},
				idsToEntryIds: nil,
			},
			args: args{
				taskId: "lol",
			},
			want: false,
		},

		{
			name: "unscheduling something that is scheduled",
			fields: fields{
				tasksService: &task.MockTasksService{},
				idsToEntryIds: map[task.RecurringTaskId]cron.EntryID{
					"hello": cron.EntryID(123),
				},
			},
			args: args{
				taskId: "hello",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &schedulerImpl{
				cron:          cron.New(cron.WithLocation(time.UTC)),
				tasksService:  tt.fields.tasksService,
				tracer:        tracing.NoopTracer{},
				idsToEntryIds: tt.fields.idsToEntryIds,
				mu:            sync.Mutex{},
				getUTC: func() time.Time {
					return time.Now().UTC()
				},
			}
			got := i.Unschedule(tt.args.taskId)
			assert.Equal(t, got, tt.want)
			assert.Empty(t, i.idsToEntryIds)
		})
	}
}

func Test_schedulerImpl_taskDefToNewTask(t *testing.T) {
	frozenNow := time.Now().UTC()
	scheduler := &schedulerImpl{
		cron:          cron.New(cron.WithLocation(time.UTC)),
		tasksService:  &task.MockTasksService{},
		tracer:        tracing.NoopTracer{},
		idsToEntryIds: make(map[task.RecurringTaskId]cron.EntryID),
		mu:            sync.Mutex{},
		getUTC: func() time.Time {
			return frozenNow
		},
	}
	recurringTaskId := task.RecurringTaskId("ttt")
	taskDef := recurring.TaskDefinition{
		Queue:             "qqq",
		RetryTimes:        123,
		Kind:              "type",
		Priority:          200,
		ProcessingTimeout: task.ProcessingTimeout(99 * time.Minute),
		Args: &task.Args{
			"aaaaarg": 1,
		},
		Context: &task.Context{
			"ctxId": 234,
		},
	}
	expectedNewTask := task.NewTask{
		Queue:             "qqq",
		RetryTimes:        123,
		Kind:              "type",
		Priority:          200,
		RunAt:             task.RunAt(frozenNow),
		ProcessingTimeout: task.ProcessingTimeout(99 * time.Minute),
		Args: &task.Args{
			"aaaaarg": 1,
		},
		Context: &task.Context{
			"ctxId": 234,
		},
		RecurringTaskId: &recurringTaskId,
	}
	result := scheduler.taskDefToNewTask(recurringTaskId, &taskDef)
	assert.EqualValues(t, &expectedNewTask, result)
}

func Test_schedulerImpl_Parse(t *testing.T) {
	scheduler := NewScheduler(&task.MockTasksService{}, tracing.NoopTracer{})
	type args struct {
		spec string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should fail for random specs",
			args: args{
				spec: "hahahahhaah",
			},
			wantErr: true,
		},
		{
			name: "should work for traditional cron specs",
			args: args{
				spec: "0,5,10 * * * *",
			},
			wantErr: false,
		},
		{
			name: "should work for macro cron specs",
			args: args{
				spec: "@every 15m",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := scheduler.Parse(tt.args.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			cronEquiv, err := cron.ParseStandard(tt.args.spec)
			assert.EqualValues(t, cronEquiv, got)
		})
	}
}

var skipIfOutstandingTaskToSchedule = recurring.Task{
	ID:                          "send-me-an-int",
	ScheduleExpression:          "@every 1s",
	SkipIfOutstandingTasksExist: true,
	TaskDefinition:              recurring.TaskDefinition{},
	IsDeleted:                   false,
	LoadedAt:                    nil,
	Metadata:                    metadata.Metadata{},
}
