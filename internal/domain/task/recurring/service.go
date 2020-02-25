package recurring

import (
	"context"
	"fmt"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
)

type Service interface {

	// Creates (persists) a NewRecurringTask, returning a RecurringTask
	//
	// An error is returned if there is already an existing RecurringTask with
	// the given id
	Create(ctx context.Context, task *NewRecurringTask) (*RecurringTask, error)

	// Retrieves a single RecurringTask, optionally returning soft-deleted tasks.
	//
	// Errors if no such RecurringTask is found that is (optionally) non-soft-deleted
	Get(ctx context.Context, id Id, includeSoftDeleted bool) (*RecurringTask, error)

	// Deletes a single RecurringTask
	//
	// Errors if no such RecurringTask is found that is non-deleted
	Delete(ctx context.Context, id Id) error

	// Loads returns all persisted RecurringTasks that have not been deleted
	//
	// Sorted by id
	All(ctx context.Context) ([]RecurringTask, error)

	// NotLoadedSince returns all non-deleted, not-loaded RecurringTasks that have been
	// modified _after_ the specific modified at time.
	//
	// Note that this accepts a ModifiedAt, but that includes RecurringTasks that were created
	// after that given time as well.
	//
	// This is used to find tasks that have been modified but not modified after a specific time
	NotLoadedSince(ctx context.Context, after metadata.ModifiedAt) ([]RecurringTask, error)

	// Updates multiple RecurringTasks at once and returns a MultiUpdateResult
	//
	// An error is returned if there is no such RecurringTask, or if there was a version conflict
	Update(ctx context.Context, update *RecurringTask) (*RecurringTask, error)

	// Updates multiple RecurringTasks at once and returns a MultiUpdateResult
	//
	// An error is returned if the update _completely_ failed.
	UpdateMultiple(ctx context.Context, updates []RecurringTask) (*MultiUpdateResult, error)
}

// MultiUpdateResult models a (partial) successful multi update result
type MultiUpdateResult struct {
	Successes        []RecurringTask
	VersionConflicts []RecurringTask
	NotFounds        []RecurringTask
	Others           []BulkUpdateOtherError
}

type BulkUpdateOtherError struct {
	RecurringTask RecurringTask
	Result        string
}

// <-- Domain Errors

// ServiceErr is an error interface for TodoRepo
type ServiceErr interface {
	error
	Id() Id
}

type WrappingErr interface {
	error
	Unwrap() error
}

// AlreadyExists is returned when the service tries to create
// a RecurringTask, but there already exists one with the same ID
type AlreadyExists struct {
	ID Id
}

func (e AlreadyExists) Error() string {
	return fmt.Sprintf("Could not find [%v]", e.ID)
}

func (e AlreadyExists) Id() Id {
	return e.ID
}

// NotFound is returned when the repo cannot find
// a repo by a given Id
type NotFound struct {
	ID Id
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

//     Errors -->
