// +build !race

// These use refreshAsNeeded, which is safely racey due to the lock+unlock from
// within the same method when accessing the last-touched queues LRU cache

package recurring

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/infra/apm/tracing"
)

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
