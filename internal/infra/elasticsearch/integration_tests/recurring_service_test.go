// +build integration

package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
	infra "github.com/lloydmeta/tasques/internal/infra/elasticsearch/task/recurring"
)

var scrollPageSize = uint(10)

func buildRecurringTasksService() recurring.Service {
	return infra.NewService(
		esClient,
		scrollPageSize,
		1*time.Minute,
	)
}

func Test_esRecurringService_verifingPersistedForm(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC()
	setRecurringTasksServiceClock(t, service, now)
	type args struct {
		toPersist *recurring.NewTask
	}
	tests := []struct {
		name       string
		args       args
		wantedJson JsonObj
	}{
		{
			name: "should not write nil fields out to ES",
			args: args{
				toPersist: &recurring.NewTask{
					ID:                 "persistence-test-1",
					ScheduleExpression: "* * * * *",
					TaskDefinition: recurring.TaskDefinition{
						Queue:             "persisted-form-test",
						RetryTimes:        1,
						Kind:              "k1",
						Priority:          2,
						ProcessingTimeout: task.ProcessingTimeout(3 * time.Second),
						Args:              nil,
						Context:           nil,
					},
				},
			},
			wantedJson: JsonObj{
				"schedule_expression": "* * * * *",
				"task_definition": JsonObj{
					"queue":              "persisted-form-test",
					"retry_times":        float64(1),
					"kind":               "k1",
					"processing_timeout": float64(3 * time.Second),
					"priority":           float64(2),
				},
				"is_deleted": false,
				"metadata": JsonObj{
					"created_at":  now.Format(time.RFC3339Nano),
					"modified_at": now.Format(time.RFC3339Nano),
				},
			},
		},
		{
			name: "should persist provided args and context",
			args: args{
				toPersist: &recurring.NewTask{
					ID:                 "persistence-test-2",
					ScheduleExpression: "* * * * 3",
					TaskDefinition: recurring.TaskDefinition{
						Queue:             "persisted-form-test",
						RetryTimes:        1,
						Kind:              "k1",
						Priority:          2,
						ProcessingTimeout: task.ProcessingTimeout(3 * time.Second),
						Args: &task.Args{
							"wut": "up",
						},
						Context: &task.Context{
							"nm": "yo",
						},
					},
				},
			},
			wantedJson: JsonObj{
				"schedule_expression": "* * * * 3",
				"task_definition": JsonObj{
					"queue":              "persisted-form-test",
					"retry_times":        float64(1),
					"kind":               "k1",
					"processing_timeout": float64(3 * time.Second),
					"priority":           float64(2),
					"args": JsonObj{
						"wut": "up",
					},
					"context": JsonObj{
						"nm": "yo",
					},
				},
				"is_deleted": false,
				"metadata": JsonObj{
					"created_at":  now.Format(time.RFC3339Nano),
					"modified_at": now.Format(time.RFC3339Nano),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			created, err := service.Create(ctx, tt.args.toPersist)
			assert.NoError(t, err)
			getReq := esapi.GetRequest{
				Index:      infra.TasquesRecurringTasksIndex,
				DocumentID: string(created.ID),
			}
			rawResp, err := getReq.Do(ctx, esClient)
			assert.NoError(t, err)
			defer rawResp.Body.Close()
			if rawResp.StatusCode == 200 {
				var resp JsonObj
				if err := json.NewDecoder(rawResp.Body).Decode(&resp); err != nil {
					assert.NoError(t, err)
				} else {
					source := resp["_source"].(JsonObj)
					//sourceProcessingTimeout := source["processing_timeout"]
					//// it's a float....
					//assert.InDelta(t, tt.wantedJson["processing_timeout"], sourceProcessingTimeout, 0.1)
					//delete(tt.wantedJson, "processing_timeout")
					//delete(source, "processing_timeout")
					assert.EqualValues(t, tt.wantedJson, source)
				}
			} else {
				t.Error("Retrieve failed from ES")
			}

		})
	}
}

func Test_esRecurringService_Create(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC()
	setRecurringTasksServiceClock(t, service, now)
	type args struct {
		toPersist *recurring.NewTask
	}
	tests := []struct {
		name  string
		setup func() error
		args  args

		buildWant func(got *recurring.Task) recurring.Task
		wantErr   bool
		errType   interface{}
	}{
		{
			name:  "create a Task",
			setup: nil,
			args: args{
				toPersist: &recurring.NewTask{
					ID:                 "create-test-1",
					ScheduleExpression: "* * * * *",
					TaskDefinition: recurring.TaskDefinition{
						Queue:             "persisted-form-test",
						RetryTimes:        1,
						Kind:              "k1",
						Priority:          2,
						ProcessingTimeout: task.ProcessingTimeout(3 * time.Second),
						Args:              nil,
						Context:           nil,
					},
				},
			},
			buildWant: func(got *recurring.Task) recurring.Task {
				return recurring.Task{
					ID:                 "create-test-1",
					ScheduleExpression: "* * * * *",
					TaskDefinition: recurring.TaskDefinition{
						Queue:             "persisted-form-test",
						RetryTimes:        1,
						Kind:              "k1",
						Priority:          2,
						ProcessingTimeout: task.ProcessingTimeout(3 * time.Second),
						Args:              nil,
						Context:           nil,
					},
					IsDeleted: false,
					LoadedAt:  nil,
					Metadata:  got.Metadata,
				}
			},
			wantErr: false,
			errType: nil,
		},
		{
			name: "should fail to create a Task when there is one with the same id that is not soft-deleted",
			setup: func() error {
				existing := JsonObj{
					"is_deleted": false,
				}
				bytesToSend, err := json.Marshal(existing)
				if err != nil {
					return err
				}
				req := esapi.CreateRequest{
					Index:      infra.TasquesRecurringTasksIndex,
					DocumentID: "create-test-2",
					Body:       bytes.NewReader(bytesToSend),
				}
				_, err = req.Do(ctx, esClient)
				if err != nil {
					return err
				}
				return nil
			},
			args: args{
				toPersist: &recurring.NewTask{
					ID:                 "create-test-2",
					ScheduleExpression: "* * * * *",
					TaskDefinition:     recurring.TaskDefinition{},
				},
			},
			buildWant: nil,
			wantErr:   true,
			errType:   recurring.AlreadyExists{},
		},
		{
			name: "should successfully create a Task when there is one with the same id that *is* soft-deleted",
			setup: func() error {
				existing := JsonObj{
					"is_deleted": true,
				}
				bytesToSend, err := json.Marshal(existing)
				if err != nil {
					return err
				}
				req := esapi.CreateRequest{
					Index:      infra.TasquesRecurringTasksIndex,
					DocumentID: "create-test-3",
					Body:       bytes.NewReader(bytesToSend),
				}
				_, err = req.Do(ctx, esClient)
				if err != nil {
					return err
				}
				return nil
			},
			args: args{
				toPersist: &recurring.NewTask{
					ID:                 "create-test-3",
					ScheduleExpression: "* * * * *",
					TaskDefinition: recurring.TaskDefinition{
						Queue:             "persisted-form-test",
						RetryTimes:        3,
						Kind:              "k4",
						Priority:          5,
						ProcessingTimeout: task.ProcessingTimeout(3 * time.Second),
						Args:              nil,
						Context:           nil,
					},
				},
			},
			buildWant: func(got *recurring.Task) recurring.Task {
				return recurring.Task{
					ID:                 "create-test-3",
					ScheduleExpression: "* * * * *",
					TaskDefinition: recurring.TaskDefinition{
						Queue:             "persisted-form-test",
						RetryTimes:        3,
						Kind:              "k4",
						Priority:          5,
						ProcessingTimeout: task.ProcessingTimeout(3 * time.Second),
						Args:              nil,
						Context:           nil,
					},
					IsDeleted: false,
					LoadedAt:  nil,
					Metadata:  got.Metadata,
				}
			},
			wantErr: false,
			errType: nil,
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tt.name)
			if tt.setup != nil {
				err := tt.setup()
				if err != nil {
					t.Errorf("Error during setup %v", err)
					return
				}
			}
			got, err := service.Create(ctx, tt.args.toPersist)
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, tt.errType, err)
			} else {
				assert.NoError(t, err)
				want := tt.buildWant(got)
				assert.EqualValues(t, &want, got)
				assert.NotEmpty(t, got.Metadata.CreatedAt)
				assert.NotEmpty(t, got.Metadata.ModifiedAt)
			}
		})
	}
}

func Test_esRecurringService_Get(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC()
	setRecurringTasksServiceClock(t, service, now)
	type args struct {
		id             recurring.Id
		includeDeleted bool
	}
	tests := []struct {
		name      string
		toPersist *recurring.NewTask
		setup     func(created *recurring.Task) error
		args      args
		wantErr   bool
		errType   interface{}
	}{
		{
			name:      "get a non-existent Task",
			toPersist: nil,
			setup:     nil,
			args: args{
				id:             "get-non-existent",
				includeDeleted: true,
			},
			wantErr: true,
			errType: recurring.NotFound{},
		},
		{
			name:      "get an existent but deleted Task with includeSoftDeleted = false",
			toPersist: nil,
			setup: func(created *recurring.Task) error {
				existing := JsonObj{
					"is_deleted": true,
				}
				bytesToSend, err := json.Marshal(existing)
				if err != nil {
					return err
				}
				req := esapi.CreateRequest{
					Index:      infra.TasquesRecurringTasksIndex,
					DocumentID: "get-existing-deleted-1",
					Body:       bytes.NewReader(bytesToSend),
				}
				_, err = req.Do(ctx, esClient)
				if err != nil {
					return err
				}
				return nil
			},
			args: args{
				id:             "get-existing-deleted-1",
				includeDeleted: false,
			},
			wantErr: true,
			errType: recurring.NotFound{},
		},
		{
			name: "get an existent but deleted Task with includeSoftDeleted = true",
			toPersist: &recurring.NewTask{
				ID:                 "get-existing-deleted-2",
				ScheduleExpression: "* * * * *",
				TaskDefinition:     recurring.TaskDefinition{},
			},
			setup: func(created *recurring.Task) error {
				existing := JsonObj{
					"doc": JsonObj{
						"is_deleted": true,
					},
				}
				bytesToSend, err := json.Marshal(existing)
				if err != nil {
					return err
				}
				req := esapi.UpdateRequest{
					Index:      infra.TasquesRecurringTasksIndex,
					DocumentID: "get-existing-deleted-2",
					Body:       bytes.NewReader(bytesToSend),
				}
				rawResp, err := req.Do(ctx, esClient)
				if err != nil {
					return err
				}
				defer rawResp.Body.Close()
				var resp common.EsUpdateResponse
				if err := json.NewDecoder(rawResp.Body).Decode(&resp); err != nil {
					return err
				}
				created.IsDeleted = true
				created.Metadata.Version = resp.Version()
				return nil
			},
			args: args{
				id:             "get-existing-deleted-2",
				includeDeleted: true,
			},
			wantErr: false,
			errType: nil,
		},
		{
			name: "get an existent but Task with includeSoftDeleted = false",
			toPersist: &recurring.NewTask{
				ID:                 "get-existing-1",
				ScheduleExpression: "* * * * *",
				TaskDefinition:     recurring.TaskDefinition{},
			},
			setup: nil,
			args: args{
				id:             "get-existing-1",
				includeDeleted: false,
			},
			wantErr: false,
			errType: nil,
		},
		{
			name: "get an existent but Task with includeSoftDeleted = true",
			toPersist: &recurring.NewTask{
				ID:                 "get-existing-2",
				ScheduleExpression: "* * * * *",
				TaskDefinition:     recurring.TaskDefinition{},
			},
			setup: nil,
			args: args{
				id:             "get-existing-2",
				includeDeleted: true,
			},
			wantErr: false,
			errType: nil,
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tt.name)
			var created *recurring.Task
			if tt.toPersist != nil {
				r, err := service.Create(ctx, tt.toPersist)
				assert.NoError(t, err)
				created = r
			}
			if tt.setup != nil {
				err := tt.setup(created)
				if err != nil {
					t.Errorf("Error during setup %v", err)
					return
				}
			}
			got, err := service.Get(ctx, tt.args.id, tt.args.includeDeleted)
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, tt.errType, err)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, created, got)
			}
		})
	}
}

func Test_esRecurringService_Delete(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC()
	setRecurringTasksServiceClock(t, service, now)
	type args struct {
		id recurring.Id
	}
	tests := []struct {
		name      string
		toPersist *recurring.NewTask
		setup     func(created *recurring.Task) error
		args      args
		wantErr   bool
		errType   interface{}
	}{
		{
			name:      "delete a non-existent Task",
			toPersist: nil,
			setup:     nil,
			args: args{
				id: "delete-non-existent",
			},
			wantErr: true,
			errType: recurring.NotFound{},
		},
		{
			name:      "delete an existent but soft-deleted Task",
			toPersist: nil,
			setup: func(created *recurring.Task) error {
				existing := JsonObj{
					"is_deleted": true,
				}
				bytesToSend, err := json.Marshal(existing)
				if err != nil {
					return err
				}
				req := esapi.CreateRequest{
					Index:      infra.TasquesRecurringTasksIndex,
					DocumentID: "delete-existing-deleted-1",
					Body:       bytes.NewReader(bytesToSend),
				}
				_, err = req.Do(ctx, esClient)
				if err != nil {
					return err
				}
				return nil
			},
			args: args{
				id: "delete-existing-deleted-1",
			},
			wantErr: true,
			errType: recurring.NotFound{},
		},
		{
			name: "delete an existent non soft-deleted Task",
			toPersist: &recurring.NewTask{
				ID:                 "delete-existing-1",
				ScheduleExpression: "* * * * *",
				TaskDefinition:     recurring.TaskDefinition{},
			},
			setup: nil,
			args: args{
				id: "delete-existing-1",
			},
			wantErr: false,
			errType: nil,
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tt.name)
			var created *recurring.Task
			if tt.toPersist != nil {
				r, err := service.Create(ctx, tt.toPersist)
				assert.NoError(t, err)
				created = r
			}
			if tt.setup != nil {
				err := tt.setup(created)
				if err != nil {
					t.Errorf("Error during setup %v", err)
					return
				}
			}
			_, err := service.Delete(ctx, tt.args.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, tt.errType, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_esRecurringService_All(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC()
	setRecurringTasksServiceClock(t, service, now)
	seedNumber := scrollPageSize * 2
	var createds []recurring.Task
	for i := uint(0); i < seedNumber; i++ {
		created, err := service.Create(ctx, &recurring.NewTask{
			ID:                 recurring.Id(fmt.Sprintf("list-test-%d", i)),
			ScheduleExpression: "* * * * *",
			TaskDefinition:     recurring.TaskDefinition{},
		})
		assert.NoError(t, err)
		createds = append(createds, *created)
	}

	refreshIndices(t)

	all, err := service.All(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, uint(len(all)), seedNumber)

	for _, listed := range all {
		assert.False(t, bool(listed.IsDeleted))
	}

	for _, seeded := range createds {
		idx := -1
		for i, listed := range all {
			if listed.ID == seeded.ID {
				idx = i
				break
			}
		}
		assert.NotEqual(t, -1, idx, "Not found in list response")
		assert.EqualValues(t, seeded, all[idx])
	}

}

func Test_esRecurringService_NotLoaded(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC().Add(15 * time.Hour) // set a future time
	setRecurringTasksServiceClock(t, service, now)
	seedNumber := scrollPageSize * 2

	var expectedAbsentIds []recurring.Id
	var expectedPresentIds []recurring.Id

	// seed deleted
	seedRecurringTasks(
		t,
		seedNumber,
		func(i uint) string {
			return fmt.Sprintf("not-loaded-since-deleted-%d", i)
		},
		true,
		now,
		nil,
		&expectedAbsentIds,
	)

	// seed loaded
	seedRecurringTasks(
		t,
		seedNumber,
		func(i uint) string {
			return fmt.Sprintf("not-loaded-since-loaded-%d", i)
		},
		false,
		now,
		&now,
		&expectedAbsentIds,
	)

	// seed undeleted, old, but unloaded
	seedRecurringTasks(
		t,
		seedNumber,
		func(i uint) string {
			return fmt.Sprintf("not-loaded-since-old-unloaded-%d", i)
		},
		false,
		now.Add(-1*time.Hour),
		nil,
		&expectedPresentIds,
	)

	// seed undeleted new and unloaded
	seedRecurringTasks(
		t,
		seedNumber,
		func(i uint) string {
			return fmt.Sprintf("not-loaded-since-new-unloaded-%d", i)
		},
		false,
		now,
		nil,
		&expectedPresentIds,
	)

	refreshIndices(t)

	notLoaded, err := service.NotLoaded(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, uint(len(notLoaded)), uint(len(expectedPresentIds)))

	for _, listed := range notLoaded {
		assert.False(t, bool(listed.IsDeleted))
		assert.Nil(t, listed.LoadedAt)
	}

	assertPresence(t, true, expectedPresentIds, notLoaded)
	assertPresence(t, false, expectedAbsentIds, notLoaded)
}

func Test_esRecurringService_Update(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC()
	setRecurringTasksServiceClock(t, service, now)

	type args struct {
		update func() *recurring.Task
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		errType interface{}
	}{
		{
			name: "update a non-existent Task",
			args: args{
				update: func() *recurring.Task {
					return &recurring.Task{
						ID: "non-existent",
						Metadata: metadata.Metadata{
							Version: metadata.Version{
								SeqNum:      100,
								PrimaryTerm: 100,
							},
						},
					}
				},
			},
			wantErr: true,
			errType: recurring.InvalidVersion{}, // ES returns this even if the id doesn't exist..
		},
		{
			name: "update a non-existent Task with wrong version",
			args: args{
				update: func() *recurring.Task {
					created, err := service.Create(ctx, &recurring.NewTask{
						ID:                 "update-test-1",
						ScheduleExpression: "* * * * *",
						TaskDefinition:     recurring.TaskDefinition{},
					})
					if err != nil {
						t.Error(err)
					}
					created.Metadata.Version = metadata.Version{
						SeqNum:      100,
						PrimaryTerm: 100,
					}
					return created
				},
			},
			wantErr: true,
			errType: recurring.InvalidVersion{},
		},
		{
			name: "update an existent Task",
			args: args{
				update: func() *recurring.Task {
					created, err := service.Create(ctx, &recurring.NewTask{
						ID:                 "update-test-2",
						ScheduleExpression: "* * * * *",
						TaskDefinition:     recurring.TaskDefinition{},
					})
					if err != nil {
						t.Error(err)
					}
					created.ScheduleExpression = "0 * * * *"
					return created
				},
			},
			wantErr: false,
			errType: nil,
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tt.name)
			updatedForm := tt.args.update()
			updated, err := service.Update(ctx, updatedForm)
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, tt.errType, err, fmt.Sprintf("%v", err))
			} else {
				assert.NoError(t, err)
				assert.Nil(t, updated.LoadedAt)
				assert.EqualValues(t, updatedForm, updated)

				retrieved, err := service.Get(ctx, updated.ID, false)
				if err != nil {
					t.Error(err)
				}
				assert.Nil(t, retrieved.LoadedAt)
				assert.EqualValues(t, updatedForm, retrieved)
			}
		})
	}
}

func Test_esRecurringService_MarkLoaded(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC()
	setRecurringTasksServiceClock(t, service, now)
	seedNumber := scrollPageSize * 2
	var createds []recurring.Task
	for i := uint(0); i < seedNumber; i++ {
		created, err := service.Create(ctx, &recurring.NewTask{
			ID:                 recurring.Id(fmt.Sprintf("mark-loaded-test-%d", i)),
			ScheduleExpression: "* * * * *",
			TaskDefinition:     recurring.TaskDefinition{},
		})
		assert.NoError(t, err)
		createds = append(createds, *created)
	}

	// Sanity check
	for _, created := range createds {
		assert.Nil(t, created.LoadedAt)
	}

	result, err := service.MarkLoaded(ctx, createds)
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, result.Successes, len(createds))
	assert.Empty(t, result.NotFounds)
	assert.Empty(t, result.VersionConflicts)
	assert.Empty(t, result.Others)

	for _, created := range createds {
		retrieved, err := service.Get(ctx, created.ID, false)
		if err != nil {
			t.Error(err)
		} else {
			assert.NotNil(t, retrieved.LoadedAt)
		}
	}
}

func setRecurringTasksServiceClock(t *testing.T, service recurring.Service, frozenTime time.Time) {
	// fast forward time on the time getter
	esService, ok := service.(*infra.EsService)
	if !ok {
		t.Error("Not esTaskService")
	}
	// Now expired
	esService.SetUTCGetter(func() time.Time {
		return frozenTime
	})
}

func refreshIndices(t *testing.T) {
	refresh := esapi.IndicesRefreshRequest{
		Index:             []string{infra.TasquesRecurringTasksIndex},
		AllowNoIndices:    esapi.BoolPtr(true),
		IgnoreUnavailable: esapi.BoolPtr(true),
	}
	_, err := refresh.Do(ctx, esClient)
	assert.NoError(t, err)
}

func seedRecurringTasks(t *testing.T, numberToSeed uint, idGenerator func(i uint) string, isDeleted bool, at time.Time, loadedAt *time.Time, appendTo *[]recurring.Id) {
	for i := uint(0); i < numberToSeed; i++ {
		existing := JsonObj{
			"schedule_expression": "* * * * 3",
			"task_definition": JsonObj{
				"queue":              "not-loaded-since-test",
				"retry_times":        float64(1),
				"kind":               "k1",
				"processing_timeout": float64(3 * time.Second),
				"priority":           float64(2),
			},
			"is_deleted": isDeleted,
			"loaded_at":  loadedAt,
			"metadata": JsonObj{
				"created_at":  at,
				"modified_at": at,
			},
		}
		bytesToSend, err := json.Marshal(existing)
		if err != nil {
			assert.NoError(t, err)
		}
		id := idGenerator(i)
		*appendTo = append(*appendTo, recurring.Id(id))
		req := esapi.CreateRequest{
			Index:      infra.TasquesRecurringTasksIndex,
			DocumentID: id,
			Body:       bytes.NewReader(bytesToSend),
		}
		_, err = req.Do(ctx, esClient)
		if err != nil {
			assert.NoError(t, err)
		}
	}

}

func assertPresence(t *testing.T, shouldBePresent bool, ids []recurring.Id, observed []recurring.Task) {
	for _, expectedPresent := range ids {
		idx := -1
		for i, listed := range observed {
			if listed.ID == expectedPresent {
				idx = i
				break
			}
		}
		if shouldBePresent {
			assert.NotEqual(t, -1, idx, "Not found but should be found")
		} else {
			assert.Equal(t, -1, idx, "Found but should not be found")
		}
	}
}
