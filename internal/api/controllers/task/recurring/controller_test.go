package recurring

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/stretchr/testify/assert"

	apiRecurring "github.com/lloydmeta/tasques/internal/api/models/task/recurring"
	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
)

func TestNew(t *testing.T) {
	type args struct {
		recurringTasksService recurring.Service
	}
	tests := []struct {
		name string
		args args
		want Controller
	}{
		{
			name: "should not panic",
			args: args{
				recurringTasksService: &recurring.MockRecurringTasksService{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { New(tt.args.recurringTasksService, cfg) })
		})
	}
}

func Test_impl_Create(t *testing.T) {
	type fields struct {
		recurringTasksService *recurring.MockRecurringTasksService
	}
	type args struct {
		task *apiRecurring.NewTask
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *apiRecurring.Task
		wantErr bool
	}{
		{
			name: "create results in already exists",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{
					CreateOverride: func() (task *recurring.Task, err error) {
						return nil, recurring.AlreadyExists{ID: "i"}
					},
				},
			},
			args: args{
				task: &apiRecurring.NewTask{
					ID:                 "t",
					ScheduleExpression: "* * * * *",
					TaskDefinition:     apiRecurring.TaskDefinition{},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "create successful",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{},
			},
			args: args{
				task: &apiRecurring.NewTask{
					ID:                 "t",
					ScheduleExpression: "* * * * *",
					TaskDefinition:     apiRecurring.TaskDefinition{},
				},
			},
			want:    &mockApiRecurringTask,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				recurringTasksService: tt.fields.recurringTasksService,
				tasksConfig:           cfg,
			}
			got, err := c.Create(ctx, tt.args.task)
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.CreateCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_impl_Update(t *testing.T) {
	type fields struct {
		recurringTasksService *recurring.MockRecurringTasksService
	}
	type args struct {
		id   task.RecurringTaskId
		task *apiRecurring.TaskUpdate
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *apiRecurring.Task
		wantErr bool
	}{
		{
			name: "not found error",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{
					UpdateOverride: func() (task *recurring.Task, err error) {
						return nil, recurring.NotFound{ID: "m"}
					},
				},
			},
			args: args{
				id:   "asdfasf",
				task: &apiRecurring.TaskUpdate{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid version error",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{
					UpdateOverride: func() (task *recurring.Task, err error) {
						return nil, recurring.InvalidVersion{ID: "m"}
					},
				},
			},
			args: args{
				id:   "asdfasf",
				task: &apiRecurring.TaskUpdate{},
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "successful, with filled in top-level fields",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{},
			},
			args: args{
				id: "asdfasf",
				task: &apiRecurring.TaskUpdate{
					ScheduleExpression: &mockApiRecurringTask.ScheduleExpression,
					TaskDefinition: &apiRecurring.TaskDefinition{
						Queue: "q",
						Kind:  "k",
					},
					SkipIfOutstandingTasksExist: esapi.BoolPtr(true),
				},
			},
			want:    &mockApiRecurringTask,
			wantErr: false,
		},
		{
			name: "successful, with empty fields",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{},
			},
			args: args{
				id:   "asdfasf",
				task: &apiRecurring.TaskUpdate{},
			},
			want:    &mockApiRecurringTask,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				recurringTasksService: tt.fields.recurringTasksService,
				tasksConfig:           cfg,
			}
			got, err := c.Update(ctx, tt.args.id, tt.args.task)
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.UpdateCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_impl_Get(t *testing.T) {
	type fields struct {
		recurringTasksService *recurring.MockRecurringTasksService
	}
	type args struct {
		id task.RecurringTaskId
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *apiRecurring.Task
		wantErr bool
	}{
		{
			name: "not found error",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{
					GetOverride: func() (task *recurring.Task, err error) {
						return nil, recurring.NotFound{ID: "m"}
					},
				},
			},
			args: args{
				id: "asdfasf",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "successful",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{},
			},
			args: args{
				id: "asdfasf",
			},
			want:    &mockApiRecurringTask,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				recurringTasksService: tt.fields.recurringTasksService,
				tasksConfig:           cfg,
			}
			got, err := c.Get(ctx, tt.args.id)
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.GetCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_impl_Delete(t *testing.T) {
	type fields struct {
		recurringTasksService *recurring.MockRecurringTasksService
	}
	type args struct {
		id task.RecurringTaskId
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *apiRecurring.Task
		wantErr bool
	}{
		{
			name: "not found error",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{
					DeleteOverride: func() (task *recurring.Task, err error) {
						return nil, recurring.NotFound{ID: "m"}
					},
				},
			},
			args: args{
				id: "asdfasf",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid version error",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{
					DeleteOverride: func() (task *recurring.Task, err error) {
						return nil, recurring.InvalidVersion{ID: "m"}
					},
				},
			},
			args: args{
				id: "asdfasf",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "successful",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{},
			},
			args: args{
				id: "asdfasf",
			},
			want:    &mockApiRecurringTask,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				recurringTasksService: tt.fields.recurringTasksService,
				tasksConfig:           cfg,
			}
			got, err := c.Delete(ctx, tt.args.id)
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.DeleteCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
		})
	}
}

func Test_impl_List(t *testing.T) {
	type fields struct {
		recurringTasksService *recurring.MockRecurringTasksService
	}
	type args struct {
		id task.RecurringTaskId
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []apiRecurring.Task
		wantErr bool
	}{
		{
			name: "bad data error",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{
					AllOverride: func() (task []recurring.Task, err error) {
						return nil, recurring.InvalidPersistedData{}
					},
				},
			},
			args: args{
				id: "asdfasf",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "successful",
			fields: fields{
				recurringTasksService: &recurring.MockRecurringTasksService{},
			},
			args: args{
				id: "asdfasf",
			},
			want:    []apiRecurring.Task{mockApiRecurringTask},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &impl{
				recurringTasksService: tt.fields.recurringTasksService,
				tasksConfig:           cfg,
			}
			got, err := c.List(ctx)
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.AllCalled)
			if err != nil && !tt.wantErr {
				t.Error(err)
			} else {
				assert.EqualValues(t, tt.want, got)
			}
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
			"NotFound errors should 404",
			args{
				recurring.NotFound{},
			},
			404,
		},
		{
			"AlreadyExists errors should 409",
			args{
				recurring.AlreadyExists{},
			},
			409,
		},
		{
			"InvalidPersistedData errors should 500",
			args{
				recurring.InvalidPersistedData{},
			},
			500,
		},
		{
			"InvalidVersion errors should 409",
			args{
				recurring.InvalidVersion{},
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

var ctx = context.Background()
var cfg = config.TasksDefaults{}
var mockApiRecurringTask = apiRecurring.FromDomainTask(&recurring.MockDomainRecurringTask)
