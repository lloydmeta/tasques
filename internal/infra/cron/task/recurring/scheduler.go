package recurring

import (
	"fmt"
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
	var cronJob cron.Job = cron.FuncJob(func() {
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

	if task.SkipIfOutstandingTasksExist {
		cronJob = cron.NewChain(
			cron.Recover(zeroLogCronLogger{}),
			cron.DelayIfStillRunning(zeroLogCronLogger{}),
		).Then(cronJob)
	} else {
		cron.NewChain(cron.Recover(zeroLogCronLogger{})).Then(cronJob)
	}

	entryId, err := i.cron.AddJob(string(task.ScheduleExpression), cronJob)
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

type zeroLogCronLogger struct {
}

func (z zeroLogCronLogger) Info(msg string, keysAndValues ...interface{}) {
	if log.Info().Enabled() {
		formatted := formatTimeValues(keysAndValues)
		log.Info().Fields(formatted).Msg(msg)
	}
}

func (z zeroLogCronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if log.Error().Enabled() {
		formatted := formatTimeValues(keysAndValues)
		log.Error().Err(err).Fields(formatted).Msg(msg)
	}
}

// formatTimeValues formats any time.Time values as RFC3339 *and*
// returns the even-odd idx key-value pair slice as a map
func formatTimeValues(keysAndValues []interface{}) map[string]interface{} {
	formattedArgs := make(map[string]interface{}, len(keysAndValues)/2)
	for idx := 0; idx < len(keysAndValues); idx += 2 {
		var key string
		if s, ok := keysAndValues[idx].(string); ok {
			key = s
		} else {
			key = fmt.Sprint(keysAndValues[idx])
		}
		valueIdx := idx + 1
		if len(keysAndValues) > valueIdx {
			value := keysAndValues[valueIdx]
			if t, ok := value.(time.Time); ok {
				value = t.Format(time.RFC3339)
			}
			formattedArgs[key] = value
		}
	}
	return formattedArgs
}
