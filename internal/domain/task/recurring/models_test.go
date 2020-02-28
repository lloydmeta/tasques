package recurring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/task"
)

func loadedAtNowUtc() *LoadedAt {
	t := LoadedAt(time.Now().UTC())
	return &t
}

func dummyMetatadata() metadata.Metadata {
	return metadata.Metadata{
		CreatedAt:  metadata.CreatedAt(time.Now().UTC()),
		ModifiedAt: metadata.ModifiedAt(time.Now().UTC()),
		Version: metadata.Version{
			SeqNum:      1,
			PrimaryTerm: 2,
		},
	}
}

func TestRecurringTask_UpdateSchedule(t *testing.T) {
	recurringTask := Task{
		ID:                 "test",
		ScheduleExpression: "* * * * *",
		TaskDefinition:     TaskDefinition{},
		IsDeleted:          false,
		LoadedAt:           loadedAtNowUtc(),
		Metadata:           dummyMetatadata(),
	}

	updatedExpression := ScheduleExpression("0 * * * *")
	recurringTask.UpdateSchedule(updatedExpression)

	assert.Equal(t, updatedExpression, recurringTask.ScheduleExpression)
	assert.Nil(t, recurringTask.LoadedAt)
}

func TestRecurringTask_UpdateTaskDefinition(t *testing.T) {
	recurringTask := Task{
		ID:                 "test",
		ScheduleExpression: "* * * * *",
		TaskDefinition:     TaskDefinition{},
		IsDeleted:          false,
		LoadedAt:           loadedAtNowUtc(),
		Metadata:           dummyMetatadata(),
	}

	updatedTaskDefinition := TaskDefinition{
		Queue:             "hello",
		RetryTimes:        1,
		Kind:              "k",
		Priority:          2,
		ProcessingTimeout: 3,
		Args: &task.Args{
			"arg": "1",
		},
		Context: &task.Context{
			"ctx": "2",
		},
	}
	recurringTask.UpdateTaskDefinition(updatedTaskDefinition)

	assert.Equal(t, updatedTaskDefinition, recurringTask.TaskDefinition)
	assert.Nil(t, recurringTask.LoadedAt)
}

func TestRecurringTask_IntoDeleted(t *testing.T) {
	recurringTask := Task{
		ID:                 "test",
		ScheduleExpression: "* * * * *",
		TaskDefinition:     TaskDefinition{},
		IsDeleted:          false,
		LoadedAt:           loadedAtNowUtc(),
		Metadata:           dummyMetatadata(),
	}

	recurringTask.IntoDeleted()
	assert.True(t, bool(recurringTask.IsDeleted))
	assert.Nil(t, recurringTask.LoadedAt)
}
