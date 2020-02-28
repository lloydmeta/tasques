package recurring

import (
	"context"
	"time"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/task"
)

var MockNow = time.Now().UTC()

type MockRecurringTasksService struct {
	CreateCalled       uint
	CreateOverride     func() (*Task, error)
	GetCalled          uint
	GetOverride        func() (*Task, error)
	UpdateCalled       uint
	UpdateOverride     func() (*Task, error)
	DeleteCalled       uint
	DeleteOverride     func() (*Task, error)
	AllCalled          uint
	AllOverride        func() ([]Task, error)
	MarkLoadedCalled   uint
	MarkLoadedOverride func() (*MultiUpdateResult, error)
	NotLoadedCalled    uint
	NotLoadedOverride  func() ([]Task, error)
}

var MockDomainRecurringTask = Task{
	ID:                 "mock",
	ScheduleExpression: "* * * * *",
	TaskDefinition: TaskDefinition{
		Queue: "q",
		Kind:  "k",
	},
	IsDeleted: false,
	LoadedAt:  nil,
	Metadata: metadata.Metadata{
		CreatedAt:  metadata.CreatedAt(MockNow),
		ModifiedAt: metadata.ModifiedAt(MockNow),
		Version: metadata.Version{
			SeqNum:      0,
			PrimaryTerm: 1,
		},
	},
}

func (m *MockRecurringTasksService) Create(ctx context.Context, task *NewTask) (*Task, error) {
	m.CreateCalled++
	if m.CreateOverride != nil {
		return m.CreateOverride()
	} else {
		return &MockDomainRecurringTask, nil
	}
}

func (m *MockRecurringTasksService) Get(ctx context.Context, id task.RecurringTaskId, includeSoftDeleted bool) (*Task, error) {
	m.GetCalled++
	if m.GetOverride != nil {
		return m.GetOverride()
	} else {
		return &MockDomainRecurringTask, nil
	}
}

func (m *MockRecurringTasksService) Delete(ctx context.Context, id task.RecurringTaskId) (*Task, error) {
	m.DeleteCalled++
	if m.DeleteOverride != nil {
		return m.DeleteOverride()
	} else {
		return &MockDomainRecurringTask, nil
	}
}

func (m *MockRecurringTasksService) All(ctx context.Context) ([]Task, error) {
	m.AllCalled++
	if m.AllOverride != nil {
		return m.AllOverride()
	} else {
		return []Task{MockDomainRecurringTask}, nil
	}
}

func (m *MockRecurringTasksService) Update(ctx context.Context, update *Task) (*Task, error) {
	m.UpdateCalled++
	if m.UpdateOverride != nil {
		return m.UpdateOverride()
	} else {
		return &MockDomainRecurringTask, nil
	}
}

func (m *MockRecurringTasksService) MarkLoaded(ctx context.Context, toMarks []Task) (*MultiUpdateResult, error) {
	m.MarkLoadedCalled++
	if m.MarkLoadedOverride != nil {
		return m.MarkLoadedOverride()
	} else {
		return &MultiUpdateResult{}, nil
	}
}

func (m *MockRecurringTasksService) NotLoaded(ctx context.Context) ([]Task, error) {
	m.NotLoadedCalled++
	if m.NotLoadedOverride != nil {
		return m.NotLoadedOverride()
	} else {
		return []Task{MockDomainRecurringTask}, nil
	}
}
