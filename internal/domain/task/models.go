package task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/worker"
)

type JsonObj map[string]interface{}

// Id for a task that has been persisted
type Id string

// Generates a random id
func GenerateId() Id {
	return Id(strings.ReplaceAll(uuid.New().String(), "-", ""))
}

// Roughly corresponds to a function name
type Kind string

// Task arguments, corresponds to the arguments of a function
type Args JsonObj

// Task context, useful if there are extra context things needed to be passed
type Context JsonObj

// When the Task should be run
type RunAt time.Time

type ProcessingTimeout time.Duration

// Priority of a task. 0 is neutral.
type Priority int

// Number of times that a Task can be retried
type RetryTimes uint

// Number of times that a Task has been retried
type AttemptedTimes uint

// Number of times that a Task can still be retried
type RemainingAttempts uint

// A Task that has yet to be created
type NewTask struct {
	Queue             queue.Name
	RetryTimes        RetryTimes
	Kind              Kind
	Priority          Priority
	RunAt             RunAt
	ProcessingTimeout ProcessingTimeout
	Args              *Args
	Context           *Context
}

// When something was last put into the queue
type EnqueuedAt time.Time

// A Task that has already been persisted
// a Task is identified uniquely by its ID and Queue and
// versioned according to its Metadata Version
type Task struct {
	ID                Id
	Queue             queue.Name
	RetryTimes        RetryTimes
	Attempted         AttemptedTimes
	Kind              Kind
	State             State
	Priority          Priority
	RunAt             RunAt
	ProcessingTimeout ProcessingTimeout
	Args              *Args
	Context           *Context
	LastClaimed       *LastClaimed
	LastEnqueuedAt    EnqueuedAt
	Metadata          metadata.Metadata
}

// RemainingAttempts returns the number of times this task can still run
// before any further updates will cause it to be marked as dead
func (t *Task) RemainingAttempts() RemainingAttempts {
	if uint(t.RetryTimes)+1 < uint(t.Attempted) {
		return 0
	} else {
		// retry times 3 (const)
		// attempted 0
		// remaining attempts:  3 + 1 - 0 = 4
		// attempted 1
		// remaining attempts:  3 + 1 - 1 = 3
		return RemainingAttempts(uint(t.RetryTimes) + 1 - uint(t.Attempted))
	}
}

// IntoClaimed marks a task as claimed.
//
// Note that it does no error checking (e.g. making sure the task is in the right status), because
// this is an internal method that is called directly after a search for claimable tasks in the
// ES tasks service implementation.
func (t *Task) IntoClaimed(workerId worker.Id, at ClaimedAt) {
	t.State = CLAIMED
	t.Attempted++
	t.LastClaimed = &LastClaimed{
		WorkerId:   workerId,
		ClaimedAt:  at,
		TimesOutAt: TimesOutAt(time.Time(at).Add(time.Duration(t.ProcessingTimeout))),
		LastReport: nil,
		Result:     nil,
	}
}

// ReportIn attaches the given NewReport to the current Task, at the given ReportedAt
// time.
//
// It does some checks to make sure, for instance, that the Task has the right status
// and currently belongs to the given worker id
func (t *Task) ReportIn(byWorker worker.Id, newReport NewReport, at ReportedAt) error {
	if t.State != CLAIMED {
		return NotClaimed{
			ID:    t.ID,
			State: t.State,
		}
	}
	if t.LastClaimed == nil || t.LastClaimed.WorkerId != byWorker {
		return NotOwnedByWorker{
			ID:                t.ID,
			WantedOwnerWorker: byWorker}
	}
	if t.LastClaimed.LastReport != nil && time.Time(t.LastClaimed.LastReport.At).After(time.Time(at)) {
		return ReportFromThePast{
			ID:                  t.ID,
			AttemptedReportTime: time.Time(at),
			ExistingReportTime:  time.Time(t.LastClaimed.LastReport.At),
		}
	}
	// Update task
	report := Report{
		At:   at,
		Data: newReport.Data,
	}
	t.LastClaimed.LastReport = &report
	t.LastClaimed.TimesOutAt = TimesOutAt(time.Time(at).Add(time.Duration(t.ProcessingTimeout)))
	return nil
}

// IntoDone marks the current task as done, attaching the given Success data
//
// It does some checks to make sure, for instance, that the Task has the right status
// and currently belongs to the given worker id
func (t *Task) IntoDone(byWorker worker.Id, at CompletedAt, success *Success) error {
	if t.State != CLAIMED {
		return NotClaimed{
			ID:    t.ID,
			State: t.State,
		}
	}
	if t.LastClaimed == nil || t.LastClaimed.WorkerId != byWorker {
		return NotOwnedByWorker{
			ID:                t.ID,
			WantedOwnerWorker: byWorker}
	}
	t.State = DONE
	t.LastClaimed.Result = &Result{
		At:      at,
		Failure: nil,
		Success: success,
	}
	return nil
}

// IntoFailed registers the current Task as something that has failed.
//
// If the Task can be retried, update its RunAt exponentially and mark it as FAILED
// If it cannot be retried, we mark it as DEAD
//
// It does some checks to make sure, for instance, that the Task has the right status
// and belongs to the given worker id
func (t *Task) IntoFailed(byWorker worker.Id, at CompletedAt, failure *Failure) error {
	if t.State != CLAIMED {
		return NotClaimed{
			ID:    t.ID,
			State: t.State,
		}
	}
	if t.LastClaimed == nil || t.LastClaimed.WorkerId != byWorker {
		return NotOwnedByWorker{
			ID:                t.ID,
			WantedOwnerWorker: byWorker}
	}

	// RetryTimes is 3
	// Attempted is 1
	// attempts_remaining is 3
	// first fail:
	//  current.RemainingAttempts > 0 (3+ 1 -1) = 3
	//  * attempted = 1
	//  * attempts_remaining = 3
	// query looks for QUEUED or FAILED
	// second fail (1st retry):
	//  current.attempts_remaining > 0 (3 + 1- 2) = 2
	//  * attempted = 2
	//  * retries_remaining = 2
	// query looks for QUEUED or FAILED
	// third fail (2nd retry):
	//  current.attempts_remaining > 0 (3 + 1 - 3) = 1
	//  * attempted = 3
	//  * retries_remaining = 1
	// query looks for QUEUED or FAILED
	// fourth fail (3rd retry):
	//  current.attempts_remaining > 0 (3- 3) = 1
	//  * marked as DEAD
	if t.RemainingAttempts() > 0 {
		// Calculate the exponential backoff before updating Attempted, taken from Faktory's formula
		// https://github.com/contribsys/faktory/wiki/Job-Errors#the-process
		var retriesSoFar int
		if t.Attempted > 0 {
			retriesSoFar = int(t.Attempted) - 1
		} else {
			retriesSoFar = 0 // impossible unless if someone played around with the storage directly, but hey
		}
		// 15 + retries ^ 4 + (rand(30) * (retries + 1))
		waitTimeSecondsInt := 15 + retriesSoFar ^ 4 + (rand.Intn(30) * (retriesSoFar + 1))
		waitTimeDuration := time.Duration(waitTimeSecondsInt * int(time.Second))
		t.RunAt = RunAt(time.Time(at).Add(waitTimeDuration))
		t.LastEnqueuedAt = EnqueuedAt(time.Time(at))
		t.State = FAILED
	} else {
		t.State = DEAD
	}
	t.LastClaimed.Result = &Result{
		At:      at,
		Failure: failure,
		Success: nil,
	}
	return nil
}

// Marks the current task as UnClaimed
//
// Decrements the "attempted" field, and requeues the Task.
//
// It does some checks to make sure, for instance, that the Task has the right status
// and belongs to the given worker id
func (t *Task) IntoUnClaimed(byWorkerId worker.Id, at EnqueuedAt) error {
	if t.State != CLAIMED {
		return NotClaimed{
			ID:    t.ID,
			State: t.State,
		}
	}
	if t.LastClaimed == nil || t.LastClaimed.WorkerId != byWorkerId {
		return NotOwnedByWorker{
			ID:                t.ID,
			WantedOwnerWorker: byWorkerId}
	}
	t.State = QUEUED
	t.Attempted--
	t.LastEnqueuedAt = at
	return nil
}

type ClaimedAt time.Time

type TimesOutAt time.Time

type Failure JsonObj
type Success JsonObj

type ReportedData JsonObj
type ReportedAt time.Time

// A Report to be filed on a Job as sent in from the owning Worker
type NewReport struct {
	Data *ReportedData
}

// A Report on a Job as sent in from the owning Worker
type Report struct {
	At   ReportedAt
	Data *ReportedData
}

type LastClaimed struct {
	WorkerId   worker.Id
	ClaimedAt  ClaimedAt
	TimesOutAt TimesOutAt
	LastReport *Report
	Result     *Result
}

type CompletedAt time.Time

type Result struct {
	At CompletedAt
	// Results. Only one of the following will be filled in at a given time
	Failure *Failure
	Success *Success
}

// Task state boilerplate galore
// The state of a Task that can be marshalled to and from JSON
type State uint8

const (
	QUEUED State = iota
	CLAIMED
	FAILED
	DONE
	DEAD

	// Do not edit these
	queued  string = "queued"
	claimed string = "claimed"
	failed  string = "failed"
	done    string = "done"
	dead    string = "dead"
)

var toString = map[State]string{
	QUEUED:  queued,
	CLAIMED: claimed,
	FAILED:  failed,
	DONE:    done,
	DEAD:    dead,
}

var toID = map[string]State{
	queued:  QUEUED,
	claimed: CLAIMED,
	failed:  FAILED,
	done:    DONE,
	dead:    DEAD,
}

func (s State) String() string {
	return toString[s]
}

// MarshalJSON marshals the enum as a quoted json string
func (s State) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(toString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *State) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	if found, ok := toID[j]; ok {
		*s = found
		return nil
	} else {
		return fmt.Errorf("invalid state: [%s]", string(b))
	}
}
