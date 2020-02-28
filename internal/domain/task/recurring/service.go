package recurring

import (
	"context"
	"fmt"

	"github.com/lloydmeta/tasques/internal/domain/task"
)

type Service interface {

	// Creates (persists) a NewTask, returning a Task
	//
	// An error is returned if there is already an existing Task with
	// the given id
	Create(ctx context.Context, task *NewTask) (*Task, error)

	// Retrieves a single Task, optionally returning soft-deleted tasks.
	//
	// Errors if no such Task is found that is (optionally) non-soft-deleted
	Get(ctx context.Context, id task.RecurringTaskId, includeSoftDeleted bool) (*Task, error)

	// Deletes a single Task
	//
	// Errors if no such Task is found that is non-deleted
	Delete(ctx context.Context, id task.RecurringTaskId) (*Task, error)

	// Loads returns all persisted RecurringTasks that have not been deleted
	//
	// Sorted by id and always reflects the realtime state of the data.
	All(ctx context.Context) ([]Task, error)

	// NotLoaded returns not-loaded (seen by recurring tasks manager) RecurringTasks.
	//
	// Note that the data returned may be not be realtime.
	//
	// This is used to find tasks that have been modified but not loaded.
	NotLoaded(ctx context.Context) ([]Task, error)

	// Updates multiple RecurringTasks at once and returns a MultiUpdateResult
	//
	// Also nils-out the LoadedAt field to make sure the persisted data works within
	// the expectations of how things are stored.
	//
	// An error is returned if there is no such Task, or if there was a version conflict
	Update(ctx context.Context, update *Task) (*Task, error)

	// MarkLoaded sets the LoadedAt field of multiple RecurringTasks to now
	// at once and returns a MultiUpdateResult.
	//
	// An error is returned if the update _completely_ failed.
	MarkLoaded(ctx context.Context, toMarks []Task) (*MultiUpdateResult, error)
}

// MultiUpdateResult models a (partial) successful multi update result
type MultiUpdateResult struct {
	Successes        []Task
	VersionConflicts []Task
	NotFounds        []Task
	Others           []BulkUpdateOtherError
}

type BulkUpdateOtherError struct {
	RecurringTask Task
	Result        string
}

// <-- Domain Errors

// ServiceErr is an error interface for Service
type ServiceErr interface {
	error
	Id() task.RecurringTaskId
}

type WrappingErr interface {
	error
	Unwrap() error
}

// AlreadyExists is returned when the service tries to create
// a Task, but there already exists one with the same ID
type AlreadyExists struct {
	ID task.RecurringTaskId
}

func (e AlreadyExists) Error() string {
	return fmt.Sprintf("Id already exists [%v]", e.ID)
}

func (e AlreadyExists) Id() task.RecurringTaskId {
	return e.ID
}

// NotFound is returned when the repo cannot find
// a repo by a given RecurringTaskId
type NotFound struct {
	ID task.RecurringTaskId
}

func (e NotFound) Error() string {
	return fmt.Sprintf("Could not find [%v]", e.ID)
}

func (e NotFound) Id() task.RecurringTaskId {
	return e.ID
}

// Invalid version returned when the version is invalid
type InvalidVersion struct {
	ID task.RecurringTaskId
}

func (e InvalidVersion) Error() string {
	return fmt.Sprintf("Could not find [%v]", e.ID)
}

func (e InvalidVersion) Id() task.RecurringTaskId {
	return e.ID
}

// Invalid data
type InvalidPersistedData struct {
	PersistedData interface{}
}

func (e InvalidPersistedData) Error() string {
	return fmt.Sprintf("Invalid persisted data: find [%v]", e.PersistedData)
}

//     Errors -->
