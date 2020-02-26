package recurring

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apiRecurring "github.com/lloydmeta/tasques/internal/api/models/task/recurring"
	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/domain/metadata"
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
				recurringTasksService: &mockRecurringTasksService{},
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
		recurringTasksService *mockRecurringTasksService
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
				recurringTasksService: &mockRecurringTasksService{
					createOverride: func() (task *recurring.Task, err error) {
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
				recurringTasksService: &mockRecurringTasksService{},
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
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.createCalled)
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
		recurringTasksService *mockRecurringTasksService
	}
	type args struct {
		id   recurring.Id
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
				recurringTasksService: &mockRecurringTasksService{
					updateOverride: func() (task *recurring.Task, err error) {
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
				recurringTasksService: &mockRecurringTasksService{
					updateOverride: func() (task *recurring.Task, err error) {
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
				recurringTasksService: &mockRecurringTasksService{},
			},
			args: args{
				id: "asdfasf",
				task: &apiRecurring.TaskUpdate{
					ScheduleExpression: &mockApiRecurringTask.ScheduleExpression,
					TaskDefinition: &apiRecurring.TaskDefinition{
						Queue: "q",
						Kind:  "k",
					},
				},
			},
			want:    &mockApiRecurringTask,
			wantErr: false,
		},
		{
			name: "successful, with empty fields",
			fields: fields{
				recurringTasksService: &mockRecurringTasksService{},
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
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.updateCalled)
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
		recurringTasksService *mockRecurringTasksService
	}
	type args struct {
		id recurring.Id
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
				recurringTasksService: &mockRecurringTasksService{
					getOverride: func() (task *recurring.Task, err error) {
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
				recurringTasksService: &mockRecurringTasksService{},
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
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.getCalled)
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
		recurringTasksService *mockRecurringTasksService
	}
	type args struct {
		id recurring.Id
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
				recurringTasksService: &mockRecurringTasksService{
					deleteOverride: func() (task *recurring.Task, err error) {
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
				recurringTasksService: &mockRecurringTasksService{
					deleteOverride: func() (task *recurring.Task, err error) {
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
				recurringTasksService: &mockRecurringTasksService{},
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
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.deleteCalled)
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
		recurringTasksService *mockRecurringTasksService
	}
	type args struct {
		id recurring.Id
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
				recurringTasksService: &mockRecurringTasksService{
					allOverride: func() (task []recurring.Task, err error) {
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
				recurringTasksService: &mockRecurringTasksService{},
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
			assert.EqualValues(t, 1, tt.fields.recurringTasksService.allCalled)
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

type mockRecurringTasksService struct {
	createCalled   uint
	createOverride func() (*recurring.Task, error)
	getCalled      uint
	getOverride    func() (*recurring.Task, error)
	updateCalled   uint
	updateOverride func() (*recurring.Task, error)
	deleteCalled   uint
	deleteOverride func() (*recurring.Task, error)
	allCalled      uint
	allOverride    func() ([]recurring.Task, error)
}

var now = time.Now().UTC()

var mockDomainRecurringTask = recurring.Task{
	ID:                 "mock",
	ScheduleExpression: "* * * * *",
	TaskDefinition: recurring.TaskDefinition{
		Queue: "q",
		Kind:  "k",
	},
	IsDeleted: false,
	LoadedAt:  nil,
	Metadata: metadata.Metadata{
		CreatedAt:  metadata.CreatedAt(now),
		ModifiedAt: metadata.ModifiedAt(now),
		Version: metadata.Version{
			SeqNum:      0,
			PrimaryTerm: 1,
		},
	},
}
var mockApiRecurringTask = apiRecurring.FromDomainTask(&mockDomainRecurringTask)

var ctx = context.Background()
var cfg = config.TasksDefaults{}

func (m *mockRecurringTasksService) Create(ctx context.Context, task *recurring.NewTask) (*recurring.Task, error) {
	m.createCalled++
	if m.createOverride != nil {
		return m.createOverride()
	} else {
		return &mockDomainRecurringTask, nil
	}
}

func (m *mockRecurringTasksService) Get(ctx context.Context, id recurring.Id, includeSoftDeleted bool) (*recurring.Task, error) {
	m.getCalled++
	if m.getOverride != nil {
		return m.getOverride()
	} else {
		return &mockDomainRecurringTask, nil
	}
}

func (m *mockRecurringTasksService) Delete(ctx context.Context, id recurring.Id) (*recurring.Task, error) {
	m.deleteCalled++
	if m.deleteOverride != nil {
		return m.deleteOverride()
	} else {
		return &mockDomainRecurringTask, nil
	}
}

func (m *mockRecurringTasksService) All(ctx context.Context) ([]recurring.Task, error) {
	m.allCalled++
	if m.allOverride != nil {
		return m.allOverride()
	} else {
		return []recurring.Task{mockDomainRecurringTask}, nil
	}
}

func (m *mockRecurringTasksService) Update(ctx context.Context, update *recurring.Task) (*recurring.Task, error) {
	m.updateCalled++
	if m.updateOverride != nil {
		return m.updateOverride()
	} else {
		return &mockDomainRecurringTask, nil
	}
}

// not exposed

func (m *mockRecurringTasksService) MarkLoaded(ctx context.Context, toMarks []recurring.Task) (*recurring.MultiUpdateResult, error) {
	panic("implement me")
}

func (m *mockRecurringTasksService) NotLoaded(ctx context.Context) ([]recurring.Task, error) {
	panic("implement me")
}
