package recurring

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
	"github.com/lloydmeta/tasques/internal/domain/tracing"
)

type schedulerImpl struct {
	cron *cron.Cron

	tasksService task.Service

	tracer tracing.Tracer

	idsToEntryIds map[task.RecurringTaskId]cron.EntryID

	mu sync.Mutex

	getUTC func() time.Time
}

// Returns the default implementation of a scheduler that delegates to
// the standard robfig/cron
func NewScheduler(tasksService task.Service, tracer tracing.Tracer) recurring.Scheduler {
	return &schedulerImpl{
		cron:          cron.New(cron.WithLocation(time.UTC)),
		tasksService:  tasksService,
		idsToEntryIds: make(map[task.RecurringTaskId]cron.EntryID),
		mu:            sync.Mutex{},
		tracer:        tracer,
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

		tx := i.tracer.BackgroundTx("recurring-task-enqueue")
		ctx := tx.Context()

		if task.SkipIfOutstandingTasksExist {
			taskQueue := task.TaskDefinition.Queue
			err := i.tasksService.RefreshAsNeeded(ctx, taskQueue)
			if err != nil {
				log.Error().
					Err(err).
					Str("id", string(task.ID)).
					Str("queue", string(taskQueue)).
					Msg("Failed to refresh Queue for Recurring Task before looking for outstanding Tasks, proceeding with search")
			}
			outstandingTasksCount, err := i.tasksService.OutstandingTasksCount(ctx, taskQueue, task.ID)
			if err != nil {
				log.Error().
					Err(err).
					Str("id", string(task.ID)).
					Str("queue", string(taskQueue)).
					Msg("Failed get count of outstanding Tasks for Recurring Task, skipping")
			} else if outstandingTasksCount > 0 {
				if log.Debug().Enabled() {
					log.Debug().
						Str("id", string(task.ID)).
						Str("queue", string(taskQueue)).
						Uint("outstandingTasks", outstandingTasksCount).
						Msg("Skipping scheduling of Recurring Task because outstanding Tasks exist")
				}
			} else {
				_, err := i.tasksService.Create(ctx, i.taskDefToNewTask(task.ID, &task.TaskDefinition))
				if err != nil {
					log.Error().
						Err(err).
						Str("id", string(task.ID)).
						Str("expression", string(task.ScheduleExpression)).
						Msg("Failed to insert new Task at interval")
				}
			}
		} else {
			_, err := i.tasksService.Create(ctx, i.taskDefToNewTask(task.ID, &task.TaskDefinition))
			if err != nil {
				log.Error().
					Err(err).
					Str("id", string(task.ID)).
					Str("expression", string(task.ScheduleExpression)).
					Msg("Failed to insert new Task at interval")
			}
		}
		tx.End()
	})
	i.idsToEntryIds[task.ID] = entryId
	return err
}

func (i *schedulerImpl) Unschedule(taskId task.RecurringTaskId) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if entryId, ok := i.idsToEntryIds[taskId]; ok {
		if log.Info().Enabled() {
			log.Info().
				Str("id", string(taskId)).
				Msg("Unscheduling recurring Task from Cron")
		}
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
	return cron.ParseStandard(spec)
}

func (i *schedulerImpl) taskDefToNewTask(id task.RecurringTaskId, def *recurring.TaskDefinition) *task.NewTask {
	return &task.NewTask{
		Queue:             def.Queue,
		RetryTimes:        def.RetryTimes,
		Kind:              def.Kind,
		Priority:          def.Priority,
		RunAt:             task.RunAt(i.getUTC()),
		ProcessingTimeout: def.ProcessingTimeout,
		Args:              def.Args,
		Context:           def.Context,
		RecurringTaskId:   &id,
	}
}
