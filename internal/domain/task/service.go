package task

import (
	"context"
	"fmt"
	"time"

	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/worker"
)

// A Service that takes care of the persistence of Tasks.
type Service interface {
	// Persists the given NewTask.
	Create(ctx context.Context, task *NewTask) (*Task, error)
	// Retrieves a Task by Id, returns an empty task with error if:
	// - No such task exists
	Get(ctx context.Context, queue queue.Name, taskId Id) (*Task, error)

	// Attempts to claim a number of Tasks for a given Worker, by worker.Id, optionally
	// blocking for the given time if the requested amount cannot be found.
	//
	// Returns immediately if the specified amount can be found, otherwise, retries until
	// the time limit, returning what it was able to claim.
	//
	// Also returns immediately if there are any errors.
	Claim(ctx context.Context, workerId worker.Id, queues []queue.Name, number uint, blockFor time.Duration) ([]Task, error)

	// Reports in on a given Task.
	//
	// Errors out if the Task
	//  1. Does not exist in the queue
	//  2. Is not claimed (possibly requeued)
	//  3. Is not claimed by the given worker as identified by id.
	//  4. Has been updated at a later time (concurrency)
	ReportIn(ctx context.Context, workerId worker.Id, queue queue.Name, taskId Id, newReport NewReport) (*Task, error)

	// Marks a Task as successfully completed.
	//
	// Errors out if the Task
	//  1. Does not exist in the queue
	//  2. Is not claimed (possibly requeued)
	//  3. Is not claimed by the given worker as identified by id.
	MarkDone(ctx context.Context, workerId worker.Id, queue queue.Name, taskId Id, success *Success) (*Task, error)

	// Marks a Task as failed.
	//
	// Errors out if the Task
	//  1. Does not exist in the queue
	//  2. Is not claimed (possibly requeued)
	//  3. Is not claimed by the given worker as identified by id.
	MarkFailed(ctx context.Context, workerId worker.Id, queue queue.Name, taskId Id, failure *Failure) (*Task, error)

	// Unclaims a Task so that it can be claimed by someone else.
	//
	// Note that we decrement attempts and reset the State, but do not unset LastClaimed; when the Task is
	// next Claimed, this field will be updated.
	//
	// Note that nothing else about the Task is modified. This allows callers to re-queue
	// a task that they have claimed but no longer wantedLastClaimed to / can handle.
	//
	// Errors out if the Task
	//  1. Does not exist in the queue
	//  2. Is not claimed (possibly requeued)
	//  3. Is not claimed by the given worker as identified by id.
	UnClaim(ctx context.Context, workerId worker.Id, queue queue.Name, taskId Id) (*Task, error)

	// This sets all Claimed Tasks that have timed out to Failed, adjusting RunAt as needed
	ReapTimedOutTasks(ctx context.Context, scrollSize uint, scrollTtl time.Duration) error
}

// <-- Domain Errors

// ServiceErr is an error interface for Service
type ServiceErr interface {
	error
	Id() Id
}

type WrappingErr interface {
	error
	Unwrap() error
}

// NotFound is returned when the repo cannot find
// a repo by a given Id
type NotFound struct {
	ID        Id
	QueueName queue.Name
}

func (e NotFound) Error() string {
	return fmt.Sprintf("Could not find [%v]", e.ID)
}

func (e NotFound) Id() Id {
	return e.ID
}

// Invalid version returned when the version is invalid
type InvalidVersion struct {
	ID Id
}

func (e InvalidVersion) Error() string {
	return fmt.Sprintf("Could not find [%v]", e.ID)
}

func (e InvalidVersion) Id() Id {
	return e.ID
}

// Invalid data
type InvalidPersistedData struct {
	PersistedData interface{}
}

func (e InvalidPersistedData) Error() string {
	return fmt.Sprintf("Invalid persisted data: find [%v]", e.PersistedData)
}

type NotOwnedByWorker struct {
	ID                Id
	WantedOwnerWorker worker.Id
}

func (n NotOwnedByWorker) Error() string {
	return fmt.Sprintf("The Task was not owned by the worker [%v]", n.WantedOwnerWorker)
}

func (n NotOwnedByWorker) Id() Id {
	return n.ID
}

type NotClaimed struct {
	ID    Id
	State State
}

func (a NotClaimed) Error() string {
	return fmt.Sprintf("The Task is not currently Claimed so cannot be reported on: [%v]", a.ID)
}

func (a NotClaimed) Id() Id {
	return a.ID
}

type ReportFromThePast struct {
	ID                  Id
	AttemptedReportTime time.Time
	ExistingReportTime  time.Time
}

func (a ReportFromThePast) Error() string {
	return fmt.Sprintf("The Task [%v] was already reported on at [%v], which overrides the current one reported at [%v]", a.ID, a.ExistingReportTime, a.AttemptedReportTime)
}

func (a ReportFromThePast) Id() Id {
	return a.ID
}

//     Errors -->
