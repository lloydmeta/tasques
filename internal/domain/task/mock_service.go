package task

import (
	"context"
	"time"

	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/worker"
)

var MockDomainTask = Task{
	ID:    "mock",
	Queue: "q",
}

type MockTasksService struct {
	CreateCalled         uint
	CreateOverride       func() (*Task, error)
	GetCalled            uint
	GetOverride          func() (*Task, error)
	ClaimCalled          uint
	ClaimOverride        func() ([]Task, error)
	ReportInCalled       uint
	ReportInOverride     func() (*Task, error)
	MarkDoneCalled       uint
	MarkDoneOverride     func() (*Task, error)
	MarkFailedCalled     uint
	MarkFailedOverride   func() (*Task, error)
	UnClaimCalled        uint
	UnClaimOverride      func() (*Task, error)
	ReapTimedOutCalled   uint
	ReapTimedOutOverride func() error
}

func (m *MockTasksService) Create(ctx context.Context, task *NewTask) (*Task, error) {
	m.CreateCalled++
	if m.CreateOverride != nil {
		return m.CreateOverride()
	} else {
		return &MockDomainTask, nil
	}
}

func (m *MockTasksService) Get(ctx context.Context, queue queue.Name, taskId Id) (*Task, error) {
	m.GetCalled++
	if m.GetOverride != nil {
		return m.GetOverride()
	} else {
		return &MockDomainTask, nil
	}
}

func (m *MockTasksService) Claim(ctx context.Context, workerId worker.Id, queues []queue.Name, number uint, blockFor time.Duration) ([]Task, error) {
	m.ClaimCalled++
	if m.ClaimOverride != nil {
		return m.ClaimOverride()
	} else {
		return []Task{MockDomainTask}, nil
	}
}

func (m *MockTasksService) ReportIn(ctx context.Context, workerId worker.Id, queue queue.Name, taskId Id, newReport NewReport) (*Task, error) {
	m.ReportInCalled++
	if m.ReportInOverride != nil {
		return m.ReportInOverride()
	} else {
		return &MockDomainTask, nil
	}
}

func (m *MockTasksService) MarkDone(ctx context.Context, workerId worker.Id, queue queue.Name, taskId Id, success *Success) (*Task, error) {
	m.MarkDoneCalled++
	if m.MarkDoneOverride != nil {
		return m.MarkDoneOverride()
	} else {
		return &MockDomainTask, nil
	}
}

func (m *MockTasksService) MarkFailed(ctx context.Context, workerId worker.Id, queue queue.Name, taskId Id, failure *Failure) (*Task, error) {
	m.MarkFailedCalled++
	if m.MarkFailedOverride != nil {
		return m.MarkFailedOverride()
	} else {
		return &MockDomainTask, nil
	}
}

func (m *MockTasksService) UnClaim(ctx context.Context, workerId worker.Id, queue queue.Name, taskId Id) (*Task, error) {
	m.UnClaimCalled++
	if m.UnClaimOverride != nil {
		return m.UnClaimOverride()
	} else {
		return &MockDomainTask, nil
	}
}

func (m *MockTasksService) ReapTimedOutTasks(ctx context.Context, scrollSize uint, scrollTtl time.Duration) error {
	m.ReapTimedOutCalled++
	if m.ReapTimedOutOverride != nil {
		return m.ReapTimedOutOverride()
	} else {
		return nil
	}
}
