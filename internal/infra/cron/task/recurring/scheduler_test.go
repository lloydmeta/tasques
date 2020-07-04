package recurring

import (
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
}

func Test_schedulerImpl_Schedule_skipIfOutstandingTasksExist(t *testing.T) {
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
			return &task.MockDomainTask, nil
		},
		OutstandingTasksCountOverride: func() (u uint, err error) {
			return 1, nil
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
	scheduler.Stop()
	time.Sleep(5 * time.Second)
	assert.GreaterOrEqual(t, uint(1), tasksService.RefreshAsNeededCalled)
	assert.GreaterOrEqual(t, uint(1), tasksService.OutstandingTasksCountCalled)
	assert.EqualValues(t, 0, tasksService.CreateCalled)
	// It should nonetheless be scheduled
	_, taskIdPresent := scheduler.idsToEntryIds[taskToSchedule.ID]
	assert.True(t, taskIdPresent)
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
