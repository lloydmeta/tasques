package integration_tests

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
	infra "github.com/lloydmeta/tasques/internal/infra/elasticsearch/task/recurring"
)

func buildRecurringTasksService() recurring.Service {
	return infra.NewService(
		esClient,
		500,
		1*time.Minute,
	)
}

func Test_esRecurringService_verifingPersistedForm(t *testing.T) {
	service := buildRecurringTasksService()
	now := time.Now().UTC()
	setRecurringTasksServiceClock(t, service, now)
	type args struct {
		toPersist *recurring.NewRecurringTask
	}
	tests := []struct {
		name       string
		args       args
		wantedJson JsonObj
	}{
		{
			name: "should not write nil fields out to ES",
			args: args{
				toPersist: &recurring.NewRecurringTask{
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
				toPersist: &recurring.NewRecurringTask{
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
		toPersist *recurring.NewRecurringTask
	}
	tests := []struct {
		name  string
		setup func() error
		args  args

		buildWant func(got *recurring.RecurringTask) recurring.RecurringTask
		wantErr   bool
		errType   interface{}
	}{
		{
			name:  "create a RecurringTask",
			setup: nil,
			args: args{
				toPersist: &recurring.NewRecurringTask{
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
			buildWant: func(got *recurring.RecurringTask) recurring.RecurringTask {
				return recurring.RecurringTask{
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
			name: "should fail to create a RecurringTask when there is one with the same id that is not soft-deleted",
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
				toPersist: &recurring.NewRecurringTask{
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
			name: "should successfully create a RecurringTask when there is one with the same id that *is* soft-deleted",
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
				toPersist: &recurring.NewRecurringTask{
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
			buildWant: func(got *recurring.RecurringTask) recurring.RecurringTask {
				return recurring.RecurringTask{
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
			if err != nil {
				assert.True(t, tt.wantErr)
				assert.IsType(t, tt.errType, err)
			} else {
				want := tt.buildWant(got)
				assert.EqualValues(t, &want, got)
			}
		})
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
