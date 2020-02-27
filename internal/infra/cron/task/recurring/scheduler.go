package recurring

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
)

// TODO tests
type schedulerImpl struct {
	cron *cron.Cron

	tasksService task.Service

	idsToEntryIds map[recurring.Id]cron.EntryID

	mu sync.Mutex

	getUTC func() time.Time
}

// Returns the default implementation of a scheduler that delegates to
// the standard robfig/cron
func NewScheduler(tasksService task.Service) recurring.Scheduler {
	return &schedulerImpl{
		cron:          cron.New(cron.WithLocation(time.UTC)),
		tasksService:  tasksService,
		idsToEntryIds: make(map[recurring.Id]cron.EntryID),
		mu:            sync.Mutex{},
		getUTC: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (i *schedulerImpl) Schedule(task recurring.Task) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if log.Info().Enabled() {
		log.Info().
			Str("id", string(task.ID)).
			Str("expression", string(task.ScheduleExpression)).
			Msg("Scheduling recurring Task to Cron")
	}

	if entryId, ok := i.idsToEntryIds[task.ID]; ok {
		i.cron.Remove(entryId)
		delete(i.idsToEntryIds, task.ID)
	}
	entryId, err := i.cron.AddFunc(string(task.ScheduleExpression), func() {
		if log.Debug().Enabled() {
			log.Debug().
				Str("id", string(task.ID)).
				Str("expression", string(task.ScheduleExpression)).
				Msg("Enqueuing Task")
		}

		_, err := i.tasksService.Create(context.Background(), i.taskDefToNewTask(&task.TaskDefinition))
		if err != nil {
			log.Error().
				Err(err).
				Str("id", string(task.ID)).
				Str("expression", string(task.ScheduleExpression)).
				Msg("Failed to insert new Task at interval")
		}
	})
	i.idsToEntryIds[task.ID] = entryId
	return err
}

func (i *schedulerImpl) Unschedule(taskId recurring.Id) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if log.Info().Enabled() {
		log.Info().
			Str("id", string(taskId)).
			Msg("Un-scheduling recurring Task from Cron")
	}

	if entryId, ok := i.idsToEntryIds[taskId]; ok {
		i.cron.Remove(entryId)
		delete(i.idsToEntryIds, taskId)
		return true
	} else {
		return false
	}
}

func (i *schedulerImpl) Start() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.cron.Start()
}

func (i *schedulerImpl) Stop() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.cron.Stop()
}
func (i *schedulerImpl) Parse(spec string) (recurring.Schedule, error) {
	return cron.ParseStandard(spec) //
}

func (i *schedulerImpl) taskDefToNewTask(def *recurring.TaskDefinition) *task.NewTask {
	return &task.NewTask{
		Queue:             def.Queue,
		RetryTimes:        def.RetryTimes,
		Kind:              def.Kind,
		Priority:          def.Priority,
		RunAt:             task.RunAt(i.getUTC()),
		ProcessingTimeout: def.ProcessingTimeout,
		Args:              def.Args,
		Context:           def.Context,
	}
}
