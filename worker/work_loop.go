package worker

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/client"
	"github.com/lloydmeta/tasques/client/tasks"
	"github.com/lloydmeta/tasques/models"
	"github.com/lloydmeta/tasques/worker/config"
)

type TaskHandle struct {
	Task          *models.TaskTask
	tasquesClient *client.Tasques
	workerId      config.WorkerId

	unclaimed uint32
}

// Submit a report with optional data to prevent the task from timing out
func (h *TaskHandle) ReportIn(data *map[string]interface{}) error {
	if !h.IsUnclaimed() {
		report := models.TaskNewReport{}
		if data != nil {
			report.Data = *data
		}
		reportParams := tasks.NewReportOnClaimedTaskParams().
			WithXTASQUESWORKERID(string(h.workerId)).
			WithQueue(*h.Task.Queue).
			WithID(*h.Task.ID).
			WithNewReport(&report)
		if _, err := h.tasquesClient.Tasks.ReportOnClaimedTask(reportParams); err != nil {
			return err
		} else {
			return nil
		}
	} else {
		return nil
	}
}

// Unclaims the current task and puts it back on the queue
func (h *TaskHandle) Unclaim() error {
	unclaimParams := tasks.NewUnclaimExistingTaskParams().
		WithXTASQUESWORKERID(string(h.workerId)).
		WithQueue(*h.Task.Queue).
		WithID(*h.Task.ID)
	if _, err := h.tasquesClient.Tasks.UnclaimExistingTask(unclaimParams); err != nil {
		return err
	} else {
		h.setUnclaimed()
		return nil
	}
}

func (h *TaskHandle) IsUnclaimed() bool {
	return atomic.LoadUint32(&h.unclaimed) > 0
}

func (h *TaskHandle) setUnclaimed() {
	atomic.StoreUint32(&h.unclaimed, 1)
}

type Success map[string]interface{}

type Failure map[string]interface{}

func (f Failure) Error() string {
	return fmt.Sprintf("Task handling resulted in failure: [%v]", map[string]interface{}(f))
}

// A Handler taks a TaskHandle and performs a function on it; the handle can be used to
// report on a given task (also bumps the timeout), unclaim the task (causes ignoring
// of the return values), or completed as either a success or failure.
type TaskHandler interface {
	Handle(handle TaskHandle) (*Success, *Failure)
}

type QueuesToHandles map[string]TaskHandler

// A work loop that submits Task claim requests to the server based on Queues and the Handlers
// that are interested in the Tasks from each Queue. Run() the returned loop to begin processing.
//
// The claim is submitted as a single request to save bandwidth, and then demuxed to the handlers
type WorkLoop struct {
	workerId         config.WorkerId
	tasquesClient    *client.Tasques
	queuesToHandles  QueuesToHandles
	claimAmount      int64
	blockFor         time.Duration
	loopStopTimeout  time.Duration
	stopSignal       uint32
	loopStopNotifier chan bool
}

func NewWorkLoop(config config.TasquesWorker, queuesToHandles QueuesToHandles) WorkLoop {
	return WorkLoop{
		workerId:         config.ID,
		tasquesClient:    config.Server.BuildClient(),
		queuesToHandles:  queuesToHandles,
		claimAmount:      int64(config.ClaimAmount),
		blockFor:         config.BlockFor,
		loopStopTimeout:  config.LoopStopTimeout,
		stopSignal:       0,
		loopStopNotifier: make(chan bool, 1),
	}
}

// Runs the work loop and listens to sig int and sig term to gracefully exit to make sure we don't
// lose any Tasks that have been claimed
func (w *WorkLoop) Run() error {
	go func() {
		// Run the loop
		if err := w.run(); err != nil {
			log.Fatal().Err(err).Msg("Failure in work loop")
		}
	}()

	// Wait for interrupt signals to gracefully shut the server down with a configurable timeout
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Work loop shutdown initialised ...")

	// Handle shutdown procedures here.
	ctx, cancel := context.WithTimeout(context.Background(), w.loopStopTimeout)
	defer cancel()
	if err := w.shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Loop did not exit in time, forcefully killing.")
	}

	log.Info().Msg("Work loop gracefully exiting")
	return nil

}

func (w *WorkLoop) run() error {

	queueNames := make([]string, 0, len(w.queuesToHandles))
	for q := range w.queuesToHandles {
		queueNames = append(queueNames, q)
	}

	claimReq := tasks.NewClaimTasksParams().
		WithXTASQUESWORKERID(string(w.workerId)).
		WithClaim(&models.TaskClaim{
			Amount:   w.claimAmount,
			BlockFor: fmt.Sprintf("%dms", w.blockFor.Milliseconds()),
			Queues:   queueNames,
		})
	for !w.isStopped() {
		claimed, err := w.tasquesClient.Tasks.ClaimTasks(claimReq)
		if err != nil {
			if apiErr, ok := err.(*runtime.APIError); ok && !(400 <= apiErr.Code && apiErr.Code <= 499) {
				log.Err(err).Msg("Server error when making a claim, ignoring")
			} else {
				return err
			}
		} else {
			for _, t := range claimed.Payload {
				handle := TaskHandle{
					Task:          t,
					tasquesClient: w.tasquesClient,
					workerId:      w.workerId,
					unclaimed:     0,
				}
				if handler, ok := w.queuesToHandles[*t.Queue]; ok {
					result, err := handler.Handle(handle)
					if !handle.IsUnclaimed() {
						if err != nil {
							if markFailedErr := w.markFailed(t, err); markFailedErr != nil {
								return err
							}
						} else if result != nil {
							if markdDoneErr := w.markSuccess(t, result); markdDoneErr != nil {
								return err
							}
						} else {
							if markFailedErr := w.markFailed(t, &Failure{"error": "No results provided on claimed Task."}); markFailedErr != nil {
								return err
							}
						}
					}
				} else {
					return fmt.Errorf("Could not find handler for queue [%v]", t.Queue)
				}
			}
		}
	}

	w.loopStopNotifier <- true
	return nil
}

func (w *WorkLoop) markFailed(t *models.TaskTask, failure *Failure) error {
	markFailedReq := tasks.NewMarkClaimedTaskFailedParams()
	markFailedReq.SetXTASQUESWORKERID(string(w.workerId))
	markFailedReq.SetQueue(*t.Queue)
	markFailedReq.SetID(*t.ID)
	markFailedReq.SetFailure(&models.TaskFailure{Data: failure})

	_, sendError := w.tasquesClient.Tasks.MarkClaimedTaskFailed(markFailedReq)
	if sendError != nil {
		log.Error().
			Interface("task", t).
			Interface("failure", failure).
			Err(sendError).
			Msg("Failed to Mark as failed.")
		return sendError
	} else {
		return nil
	}
}
func (w *WorkLoop) markSuccess(t *models.TaskTask, result *Success) error {
	markDoneReq := tasks.NewMarkClaimedTaskDoneParams()
	markDoneReq.SetXTASQUESWORKERID(string(w.workerId))
	markDoneReq.SetQueue(*t.Queue)
	markDoneReq.SetID(*t.ID)
	markDoneReq.SetSuccess(&models.TaskSuccess{Data: result})
	_, sendError := w.tasquesClient.Tasks.MarkClaimedTaskDone(markDoneReq)
	if sendError != nil {
		log.Error().
			Interface("task", t).
			Interface("result", result).
			Err(sendError).
			Msg("Failed to Mark as done.")
		return sendError
	} else {
		return nil
	}
}

func (w *WorkLoop) isStopped() bool {
	return atomic.LoadUint32(&w.stopSignal) > 0
}

func (w *WorkLoop) stop() {
	atomic.StoreUint32(&w.stopSignal, 1)
}

func (w *WorkLoop) shutdown(ctx context.Context) error {
	w.stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.loopStopNotifier:
		return nil
	}
}
