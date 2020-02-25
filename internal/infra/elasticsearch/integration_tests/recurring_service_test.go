package integration_tests

import (
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
