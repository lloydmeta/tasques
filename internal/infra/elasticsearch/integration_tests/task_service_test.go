// +build integration

package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/worker"
	task2 "github.com/lloydmeta/tasques/internal/infra/elasticsearch/task"
)

func buildTasksService() task.Service {
	return task2.NewService(
		esClient,
		config.TasksDefaults{
			BlockFor:                    3 * time.Second,
			BlockForRetryMinWait:        10 * time.Millisecond,
			BlockForRetryMaxRetries:     100,
			WorkerProcessingTimeout:     15 * time.Minute,
			ClaimAmount:                 5,
			ClaimAmountSearchMultiplier: 5,
			RetryTimes:                  50,
			VersionConflictRetryTimes:   500,
		},
	)
}

var ctx = context.Background()

type JsonObj = map[string]interface{}

func Test_esTaskService_Create_verifingPersistedForm(t *testing.T) {
	service := buildTasksService()
	now := time.Now().UTC()
	setTasksServiceClock(t, service, now)
	type args struct {
		task *task.NewTask
	}
	tests := []struct {
		name       string
		args       args
		wantedJson JsonObj
	}{
		{
			name: "should not write nil fields out to ES",
			args: args{
				task: &task.NewTask{
					Queue:             "persisted-form-test",
					RetryTimes:        1,
					Kind:              "k1",
					Priority:          2,
					RunAt:             task.RunAt(now),
					ProcessingTimeout: task.ProcessingTimeout(3 * time.Second),
					Args:              nil,
					Context:           nil,
				},
			},
			wantedJson: JsonObj{
				"retry_times":        float64(1),
				"remaining_attempts": float64(2),
				"kind":               "k1",
				"priority":           float64(2),
				"state":              "queued",
				"run_at":             now.Format(time.RFC3339Nano),
				"processing_timeout": float64(3 * time.Second),
				"last_enqueued_at":   now.Format(time.RFC3339Nano),
				"metadata": JsonObj{
					"created_at":  now.Format(time.RFC3339Nano),
					"modified_at": now.Format(time.RFC3339Nano),
				},
			},
		},
		{
			name: "should persist provided args and context",
			args: args{
				task: &task.NewTask{
					Queue:             "persisted-form-test",
					RetryTimes:        2,
					Kind:              "k1",
					Priority:          3,
					RunAt:             task.RunAt(now),
					ProcessingTimeout: task.ProcessingTimeout(4 * time.Second),
					Args: &task.Args{
						"hello": "world",
					},
					Context: &task.Context{
						"hallo": "welt",
					},
				},
			},
			wantedJson: JsonObj{
				"retry_times":        float64(2),
				"remaining_attempts": float64(3),
				"kind":               "k1",
				"priority":           float64(3),
				"state":              "queued",
				"run_at":             now.Format(time.RFC3339Nano),
				"processing_timeout": float64(4 * time.Second),
				"last_enqueued_at":   now.Format(time.RFC3339Nano),
				"args": JsonObj{
					"hello": "world",
				},
				"context": JsonObj{
					"hallo": "welt",
				},
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
			created, err := service.Create(ctx, tt.args.task)
			assert.NoError(t, err)
			getReq := esapi.GetRequest{
				Index:      string(task2.BuildIndexName(created.Queue)),
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

func Test_esTaskService_Create(t *testing.T) {
	service := buildTasksService()
	runAt := task.RunAt(time.Now().UTC())
	type args struct {
		task *task.NewTask
	}
	tests := []struct {
		name      string
		args      args
		buildWant func(got *task.Task) task.Task
		wantErr   bool
	}{
		{
			"create a task",
			args{
				&task.NewTask{
					Queue:             "anywhere-but-here",
					RetryTimes:        0,
					Priority:          task.Priority(0),
					Kind:              task.Kind("justATest"),
					ProcessingTimeout: task.ProcessingTimeout(1 * time.Hour),
					RunAt:             runAt,
					Args: &task.Args{
						"something":    "something",
						"anotherThing": "something",
					},
					Context: &task.Context{
						"reqId": "foobarhogehoge",
					},
				},
			},

			func(got *task.Task) task.Task {
				return task.Task{
					ID:                got.ID,
					Queue:             "anywhere-but-here",
					RetryTimes:        0,
					Priority:          got.Priority,
					Attempted:         0,
					Kind:              task.Kind("justATest"),
					State:             task.QUEUED,
					RunAt:             runAt,
					ProcessingTimeout: task.ProcessingTimeout(1 * time.Hour),
					Args: &task.Args{
						"something":    "something",
						"anotherThing": "something",
					},
					Context: &task.Context{
						"reqId": "foobarhogehoge",
					},
					LastClaimed:    nil,
					LastEnqueuedAt: got.LastEnqueuedAt,
					Metadata:       got.Metadata,
				}
			},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := service.Create(ctx, tt.args.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			want := tt.buildWant(got)
			assert.EqualValues(t, &want, got)
		})
	}
}

func Test_esTaskService_Get(t *testing.T) {
	service := buildTasksService()
	runAt := task.RunAt(time.Now().UTC())

	toCreate := task.NewTask{
		Queue:             "somewhere-out-there",
		RetryTimes:        0,
		Priority:          0,
		Kind:              task.Kind("justATest"),
		ProcessingTimeout: task.ProcessingTimeout(1 * time.Hour),
		RunAt:             runAt,
		Args: &task.Args{
			"something":    "something",
			"anotherThing": "something",
		},
		Context: &task.Context{
			"reqId": "abc123",
		},
	}

	created, err := service.Create(ctx, &toCreate)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		queue  queue.Name
		taskId task.Id
	}
	tests := []struct {
		name string

		args    args
		want    *task.Task
		wantErr bool
	}{
		{
			"get a non-existent task",
			args{
				queue.Name("lolololasdf123"),
				task.Id("lolololasdf123"),
			},
			nil,
			true,
		},
		{
			"get an existent task",
			args{
				created.Queue,
				created.ID,
			},
			created,
			false,
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := service.Get(ctx, tt.args.queue, tt.args.taskId)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			want := tt.want
			assert.EqualValues(t, want, got)
		})
	}
}

func Test_esTaskService_Claim(t *testing.T) {
	service := buildTasksService()

	blockFor := 3 * time.Second

	type args struct {
		queuesToSeed        []queue.Name
		tasksToSeedPerQueue int
		workerId            worker.Id
		queuesToClaimFrom   []queue.Name
		numberToClaim       uint
	}
	tests := []struct {
		name string

		args            args
		expectedClaimed int
		wantErr         bool
	}{
		{
			name: "claiming from queues with no Tasks",
			args: args{
				queuesToSeed:        nil,
				tasksToSeedPerQueue: 0,
				workerId:            "worker",
				queuesToClaimFrom:   []queue.Name{"none", "nope", "never"},
				numberToClaim:       0,
			},
			expectedClaimed: 0,
			wantErr:         false,
		},
		{
			name: "claiming _some_ from queues with Tasks",
			args: args{
				queuesToSeed:        []queue.Name{"claim-queue-1", "claim-queue-2"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker",
				queuesToClaimFrom:   []queue.Name{"claim-queue-1", "claim-queue-2"},
				numberToClaim:       10,
			},
			expectedClaimed: 10,
			wantErr:         false,
		},
		{
			name: "claiming all from queues with Tasks",
			args: args{
				queuesToSeed:        []queue.Name{"claim-queue-3", "claim-queue-4"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker",
				queuesToClaimFrom:   []queue.Name{"claim-queue-3", "claim-queue-4"},
				numberToClaim:       20,
			},
			expectedClaimed: 20,
			wantErr:         false,
		},
		{
			name: "claiming more than there are Tasks from queues with Tasks",
			args: args{
				queuesToSeed:        []queue.Name{"claim-queue-5", "claim-queue-6"},
				tasksToSeedPerQueue: 3,
				workerId:            "worker",
				queuesToClaimFrom:   []queue.Name{"claim-queue-5", "claim-queue-6"},
				numberToClaim:       20,
			},
			expectedClaimed: 6,
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, queueToSeed := range tt.args.queuesToSeed {
				_ = seedTasks(t, service, tt.args.tasksToSeedPerQueue, queueToSeed)
			}

			claimed, err := service.Claim(ctx, tt.args.workerId, tt.args.queuesToClaimFrom, tt.args.numberToClaim, blockFor)

			if (err != nil) != tt.wantErr {
				t.Errorf("Claim() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else {
				assert.Len(t, claimed, tt.expectedClaimed)
				for _, claimedTask := range claimed {
					assert.EqualValues(t, 1, claimedTask.Attempted)
					retreived, err := service.Get(ctx, claimedTask.Queue, claimedTask.ID)
					if err != nil {
						t.Error(err)
					}
					assert.EqualValues(t, 1, retreived.Attempted)
					assert.EqualValues(t, task.CLAIMED, retreived.State)
				}
			}
		})
	}
}

func Test_esTaskService_Claim_with_parallel_completing_claimers(t *testing.T) {
	service := buildTasksService()

	blockFor := 3 * time.Second

	tasksToSeed := 10

	queueName := queue.Name("competing_parallel_claimers")

	seededTasks := seedTasks(t, service, tasksToSeed, queueName)
	assert.Len(t, seededTasks, tasksToSeed)

	var waitGroup sync.WaitGroup

	claimers := 10

	// buffered
	claimedChannel := make(chan task.Task, tasksToSeed*claimers)

	for i := 0; i < claimers; i++ {
		waitGroup.Add(1)
		go func(workerIdx int) {
			workerId := worker.Id(fmt.Sprintf("paralle-worker-%d", workerIdx))
			claimed, err := service.Claim(ctx, workerId, []queue.Name{queueName}, 10, blockFor)
			assert.NoError(t, err)
			for _, c := range claimed {
				claimedChannel <- c
			}
			waitGroup.Done()
		}(i)
	}

	waitGroup.Wait()
	close(claimedChannel)

	var totalClaimed []task.Task
	for c := range claimedChannel {
		totalClaimed = append(totalClaimed, c)
	}

	assert.Len(t, totalClaimed, tasksToSeed)
}

func taskIdPtr(taskId task.Id) *task.Id {
	return &taskId
}
func queuePtr(name queue.Name) *queue.Name {
	return &name
}

func Test_esTaskService_ReportIn(t *testing.T) {

	blockFor := 3 * time.Second
	newReport := task.NewReport{Data: &task.ReportedData{
		"something": "happened",
	}}

	type args struct {
		queuesToSeed        []queue.Name
		tasksToSeedPerQueue int
		workerId            worker.Id
		reportWithWorkerId  worker.Id
		queuesToClaimFrom   []queue.Name
		numberToClaim       uint

		taskFromCreated func(tasks []task.Task) task.Task

		forceQueue   *queue.Name
		forceClaimId *task.Id
	}

	tests := []struct {
		name string

		args            args
		beforeClaimTest func(service *task2.EsService, claimed []task.Task)
		check           func(result *task.Task, err error)
	}{
		{
			name: "reporting on a non-existent task",
			args: args{
				queuesToSeed:        nil,
				tasksToSeedPerQueue: 0,
				workerId:            "worker",
				reportWithWorkerId:  "worker",
				queuesToClaimFrom:   []queue.Name{"none", "nope", "never"},
				forceClaimId:        taskIdPtr("abc123"),
				forceQueue:          queuePtr("abc123"),
			},
			check: func(result *task.Task, err error) {
				var expected task.NotFound
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "reporting on a task that isn't claimed",
			args: args{
				queuesToSeed:        []queue.Name{"report-queue-1", "report-queue-2"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker",
				reportWithWorkerId:  "worker",
				queuesToClaimFrom:   []queue.Name{"report-queue-1", "report-queue-2"},
				numberToClaim:       0,
				taskFromCreated: func(tasks []task.Task) task.Task {
					return tasks[0]
				},
			},
			check: func(result *task.Task, err error) {
				var expected task.NotClaimed
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "reporting on a task that doesn't belong to the current worker",
			args: args{
				queuesToSeed:        []queue.Name{"report-queue-3", "report-queue-4"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker1",
				reportWithWorkerId:  "worker2",
				queuesToClaimFrom:   []queue.Name{"report-queue-3", "report-queue-4"},
				numberToClaim:       10,
			},

			check: func(result *task.Task, err error) {
				var expected task.NotOwnedByWorker
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "reporting on a task that belongs to the current worker but has been reported on in the future",
			args: args{
				queuesToSeed:        []queue.Name{"report-queue-5", "report-queue-6"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker1",
				reportWithWorkerId:  "worker1",
				queuesToClaimFrom:   []queue.Name{"report-queue-51", "report-queue-6"},
				numberToClaim:       10,
			},
			beforeClaimTest: func(service *task2.EsService, claimed []task.Task) {
				toClaim := claimed[0]
				_, err := service.ReportIn(ctx, "worker1", toClaim.Queue, toClaim.ID, newReport)
				if err != nil {
					t.Error(err)
				}
				service.SetUTCGetter(func() time.Time {
					return time.Now().Add(-1 * time.Hour)
				})
			},
			check: func(result *task.Task, err error) {
				var expected task.ReportFromThePast
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "reporting on a task that belongs to the current worker",
			args: args{
				queuesToSeed:        []queue.Name{"report-queue-5", "report-queue-6"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker1",
				reportWithWorkerId:  "worker1",
				queuesToClaimFrom:   []queue.Name{"report-queue-51", "report-queue-6"},
				numberToClaim:       10,
			},
			check: func(result *task.Task, err error) {
				assert.Nil(t, err)
				assert.EqualValues(t, newReport.Data, result.LastClaimed.LastReport.Data)
			},
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var seeded []task.Task
			service := buildTasksService()
			for _, queueToSeed := range tt.args.queuesToSeed {
				created := seedTasks(t, service, tt.args.tasksToSeedPerQueue, queueToSeed)
				seeded = append(seeded, created...)

			}

			claimed, err := service.Claim(ctx, tt.args.workerId, tt.args.queuesToClaimFrom, tt.args.numberToClaim, blockFor)
			if err != nil {
				t.Error(err)
			}

			if tt.beforeClaimTest != nil {
				if esService, ok := service.(*task2.EsService); ok {
					tt.beforeClaimTest(esService, claimed)
				}
			}

			if tt.args.forceClaimId != nil && tt.args.forceQueue != nil {
				result, err := service.ReportIn(ctx, tt.args.reportWithWorkerId, *tt.args.forceQueue, *tt.args.forceClaimId, newReport)
				tt.check(result, err)
			} else if tt.args.taskFromCreated != nil {
				t := tt.args.taskFromCreated(seeded)
				result, err := service.ReportIn(ctx, tt.args.reportWithWorkerId, t.Queue, t.ID, newReport)
				tt.check(result, err)
			} else {
				// Assume we have non-empty claims
				claimedTask := claimed[0]

				result, err := service.ReportIn(ctx, tt.args.reportWithWorkerId, claimedTask.Queue, claimedTask.ID, newReport)
				tt.check(result, err)
				if err == nil {
					retrieved, err := service.Get(ctx, claimedTask.Queue, claimedTask.ID)
					if err != nil {
						t.Error(err)
					} else {
						assert.EqualValues(t, retrieved.LastClaimed.LastReport.Data, newReport.Data)
						assert.GreaterOrEqual(t, time.Time(retrieved.LastClaimed.TimesOutAt).UnixNano(), time.Time(claimedTask.LastClaimed.TimesOutAt).UnixNano())
					}
				}
			}
		})
	}
}

func Test_esTaskService_MarkDone(t *testing.T) {

	blockFor := 3 * time.Second
	success := task.Success{"message": "it worked"}

	type args struct {
		queuesToSeed        []queue.Name
		tasksToSeedPerQueue int
		claimUserId         worker.Id
		markDownWorkerId    worker.Id
		queuesToClaimFrom   []queue.Name
		numberToClaim       uint

		taskFromCreated func(tasks []task.Task) task.Task

		forceQueue   *queue.Name
		forceClaimId *task.Id
	}

	tests := []struct {
		name string

		args               args
		beforeMarkComplete func(service *task2.EsService, claimed []task.Task)
		check              func(result *task.Task, err error)
	}{
		{
			name: "marking a non-existent task as done",
			args: args{
				queuesToSeed:        nil,
				tasksToSeedPerQueue: 0,
				claimUserId:         "worker",
				markDownWorkerId:    "worker",
				queuesToClaimFrom:   []queue.Name{"none", "nope", "never"},
				forceClaimId:        taskIdPtr("abc123"),
				forceQueue:          queuePtr("abc123"),
			},
			check: func(result *task.Task, err error) {
				var expected task.NotFound
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "marking a task that isn't claimed as done",
			args: args{
				queuesToSeed:        []queue.Name{"mark-done-1", "mark-done-2"},
				tasksToSeedPerQueue: 10,
				claimUserId:         "worker",
				markDownWorkerId:    "worker",
				queuesToClaimFrom:   []queue.Name{"mark-done-1", "mark-done-2"},
				numberToClaim:       0,
				taskFromCreated: func(tasks []task.Task) task.Task {
					return tasks[0]
				},
			},
			check: func(result *task.Task, err error) {
				var expected task.NotClaimed
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "marking a task that doesn't belong to the current worker as done",
			args: args{
				queuesToSeed:        []queue.Name{"mark-done-3", "mark-done-4"},
				tasksToSeedPerQueue: 10,
				claimUserId:         "worker1",
				markDownWorkerId:    "worker2",
				queuesToClaimFrom:   []queue.Name{"mark-done-3", "mark-done-4"},
				numberToClaim:       10,
			},

			check: func(result *task.Task, err error) {
				var expected task.NotOwnedByWorker
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "marking a task that belongs to the current worker as done but has already been marked as complete",
			args: args{
				queuesToSeed:        []queue.Name{"mark-done-5", "mark-done-6"},
				tasksToSeedPerQueue: 10,
				claimUserId:         "worker1",
				markDownWorkerId:    "worker1",
				queuesToClaimFrom:   []queue.Name{"mark-done-51", "mark-done-6"},
				numberToClaim:       10,
			},
			beforeMarkComplete: func(service *task2.EsService, claimed []task.Task) {
				toClaim := claimed[0]
				_, err := service.MarkDone(ctx, "worker1", toClaim.Queue, toClaim.ID, nil)
				if err != nil {
					t.Error(err)
				}
			},
			check: func(result *task.Task, err error) {
				var expected task.NotClaimed
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "marking a task claimed by the current user as done",
			args: args{
				queuesToSeed:        []queue.Name{"mark-done-5", "mark-done-6"},
				tasksToSeedPerQueue: 10,
				claimUserId:         "worker1",
				markDownWorkerId:    "worker1",
				queuesToClaimFrom:   []queue.Name{"mark-done-51", "mark-done-6"},
				numberToClaim:       10,
			},
			check: func(result *task.Task, err error) {
				assert.Nil(t, err)
				assert.Nil(t, result.LastClaimed.Result.Failure)
				assert.EqualValues(t, &success, result.LastClaimed.Result.Success)
			},
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var seeded []task.Task
			service := buildTasksService()
			for _, queueToSeed := range tt.args.queuesToSeed {
				created := seedTasks(t, service, tt.args.tasksToSeedPerQueue, queueToSeed)
				seeded = append(seeded, created...)
			}

			claimed, err := service.Claim(ctx, tt.args.claimUserId, tt.args.queuesToClaimFrom, tt.args.numberToClaim, blockFor)
			if err != nil {
				t.Error(err)
			}

			if tt.beforeMarkComplete != nil {
				if esService, ok := service.(*task2.EsService); ok {
					tt.beforeMarkComplete(esService, claimed)
				}
			}

			if tt.args.forceClaimId != nil && tt.args.forceQueue != nil {
				result, err := service.MarkDone(ctx, tt.args.markDownWorkerId, *tt.args.forceQueue, *tt.args.forceClaimId, &success)
				tt.check(result, err)
			} else if tt.args.taskFromCreated != nil {
				t := tt.args.taskFromCreated(seeded)
				result, err := service.MarkDone(ctx, tt.args.markDownWorkerId, t.Queue, t.ID, &success)
				tt.check(result, err)
			} else {
				// Assume we have non-empty claims
				claimedTask := claimed[0]
				result, err := service.MarkDone(ctx, tt.args.markDownWorkerId, claimedTask.Queue, claimedTask.ID, &success)
				tt.check(result, err)
				if err == nil {
					retrieved, err := service.Get(ctx, claimedTask.Queue, claimedTask.ID)
					if err != nil {
						t.Error(err)
					} else {
						assert.EqualValues(t, retrieved.State, task.DONE)
						assert.Nil(t, result.LastClaimed.Result.Failure)
						assert.EqualValues(t, retrieved.LastClaimed.Result.Success, &success)
					}
				}
			}
		})
	}
}

func Test_esTaskService_MarkFailed(t *testing.T) {

	blockFor := 3 * time.Second
	failure := task.Failure{"message": "it worked"}

	type args struct {
		queuesToSeed        []queue.Name
		tasksToSeedPerQueue int
		claimUserId         worker.Id
		markFailedUserId    worker.Id
		queuesToClaimFrom   []queue.Name
		numberToClaim       uint

		taskFromCreated func(tasks []task.Task) task.Task

		forceQueue   *queue.Name
		forceClaimId *task.Id
	}

	tests := []struct {
		name string

		args               args
		beforeMarkComplete func(service *task2.EsService, claimed []task.Task)
		check              func(result *task.Task, err error)
	}{
		{
			name: "marking a non-existent task as failed",
			args: args{
				queuesToSeed:        nil,
				tasksToSeedPerQueue: 0,
				claimUserId:         "worker",
				markFailedUserId:    "worker",
				queuesToClaimFrom:   []queue.Name{"none", "nope", "never"},
				forceClaimId:        taskIdPtr("abc123"),
				forceQueue:          queuePtr("abc123"),
			},
			check: func(result *task.Task, err error) {
				var expected task.NotFound
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "marking a task that isn't claimed as failed",
			args: args{
				queuesToSeed:        []queue.Name{"mark-failed-1", "mark-failed-2"},
				tasksToSeedPerQueue: 10,
				claimUserId:         "worker",
				markFailedUserId:    "worker",
				queuesToClaimFrom:   []queue.Name{"mark-failed-1", "mark-failed-2"},
				numberToClaim:       0,
				taskFromCreated: func(tasks []task.Task) task.Task {
					return tasks[0]
				},
			},
			check: func(result *task.Task, err error) {
				var expected task.NotClaimed
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "marking a task that doesn't belong to the current worker as failed",
			args: args{
				queuesToSeed:        []queue.Name{"mark-failed-3", "mark-failed-4"},
				tasksToSeedPerQueue: 10,
				claimUserId:         "worker1",
				markFailedUserId:    "worker2",
				queuesToClaimFrom:   []queue.Name{"mark-failed-3", "mark-failed-4"},
				numberToClaim:       10,
			},

			check: func(result *task.Task, err error) {
				var expected task.NotOwnedByWorker
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "marking a task that belongs to the current worker as done but has already been marked as complete",
			args: args{
				queuesToSeed:        []queue.Name{"mark-failed-5", "mark-failed-6"},
				tasksToSeedPerQueue: 10,
				claimUserId:         "worker1",
				markFailedUserId:    "worker1",
				queuesToClaimFrom:   []queue.Name{"mark-failed-51", "mark-failed-6"},
				numberToClaim:       10,
			},
			beforeMarkComplete: func(service *task2.EsService, claimed []task.Task) {
				toClaim := claimed[0]
				_, err := service.MarkDone(ctx, "worker1", toClaim.Queue, toClaim.ID, nil)
				if err != nil {
					t.Error(err)
				}
			},
			check: func(result *task.Task, err error) {
				var expected task.NotClaimed
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "marking a task claimed by the current user as failed",
			args: args{
				queuesToSeed:        []queue.Name{"mark-failed-5", "mark-failed-6"},
				tasksToSeedPerQueue: 10,
				claimUserId:         "worker1",
				markFailedUserId:    "worker1",
				queuesToClaimFrom:   []queue.Name{"mark-failed-51", "mark-failed-6"},
				numberToClaim:       10,
			},
			check: func(result *task.Task, err error) {
				assert.Nil(t, err)
				assert.Nil(t, result.LastClaimed.Result.Success)
				assert.EqualValues(t, &failure, result.LastClaimed.Result.Failure)
			},
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var seeded []task.Task
			service := buildTasksService()
			for _, queueToSeed := range tt.args.queuesToSeed {
				created := seedTasks(t, service, tt.args.tasksToSeedPerQueue, queueToSeed)
				seeded = append(seeded, created...)
			}

			claimed, err := service.Claim(ctx, tt.args.claimUserId, tt.args.queuesToClaimFrom, tt.args.numberToClaim, blockFor)
			if err != nil {
				t.Error(err)
			}

			if tt.beforeMarkComplete != nil {
				if esService, ok := service.(*task2.EsService); ok {
					tt.beforeMarkComplete(esService, claimed)
				}
			}

			if tt.args.forceClaimId != nil && tt.args.forceQueue != nil {
				result, err := service.MarkFailed(ctx, tt.args.markFailedUserId, *tt.args.forceQueue, *tt.args.forceClaimId, &failure)
				tt.check(result, err)
			} else if tt.args.taskFromCreated != nil {
				t := tt.args.taskFromCreated(seeded)
				result, err := service.MarkFailed(ctx, tt.args.markFailedUserId, t.Queue, t.ID, &failure)
				tt.check(result, err)
			} else {
				// Assume we have non-empty claims
				claimedTask := claimed[0]
				result, err := service.MarkFailed(ctx, tt.args.markFailedUserId, claimedTask.Queue, claimedTask.ID, &failure)
				tt.check(result, err)
				if err == nil {
					retrieved, err := service.Get(ctx, claimedTask.Queue, claimedTask.ID)
					if err != nil {
						t.Error(err)
					} else {
						assert.EqualValues(t, retrieved.State, task.FAILED)
						assert.Nil(t, retrieved.LastClaimed.Result.Success)
						assert.EqualValues(t, retrieved.LastClaimed.Result.Failure, &failure)
					}
				}
			}
		})
	}
}

func Test_esTaskService_UnClaim(t *testing.T) {

	blockFor := 3 * time.Second

	type args struct {
		queuesToSeed        []queue.Name
		tasksToSeedPerQueue int
		workerId            worker.Id
		unclaimWithWorker   worker.Id
		queuesToClaimFrom   []queue.Name
		numberToClaim       uint

		taskFromCreated func(tasks []task.Task) task.Task

		forceQueue   *queue.Name
		forceClaimId *task.Id
	}

	tests := []struct {
		name string

		args          args
		beforeUnclaim func(service *task2.EsService, claimed []task.Task)
		check         func(result *task.Task, err error)
	}{
		{
			name: "unclaiming a non-existent task",
			args: args{
				queuesToSeed:        nil,
				tasksToSeedPerQueue: 0,
				workerId:            "worker",
				unclaimWithWorker:   "worker",
				queuesToClaimFrom:   []queue.Name{"none", "nope", "never"},
				forceClaimId:        taskIdPtr("abc123"),
				forceQueue:          queuePtr("abc123"),
			},
			check: func(result *task.Task, err error) {
				var expected task.NotFound
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "unclaiming a task that isn't claimed",
			args: args{
				queuesToSeed:        []queue.Name{"unclaim-queue-1", "unclaim-queue-2"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker",
				unclaimWithWorker:   "worker",
				queuesToClaimFrom:   []queue.Name{"unclaim-queue-1", "unclaim-queue-2"},
				numberToClaim:       0,
				taskFromCreated: func(tasks []task.Task) task.Task {
					return tasks[0]
				},
			},
			check: func(result *task.Task, err error) {
				var expected task.NotClaimed
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "unclaiming a task that doesn't belong to the current worker",
			args: args{
				queuesToSeed:        []queue.Name{"unclaim-queue-3", "unclaim-queue-4"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker1",
				unclaimWithWorker:   "worker2",
				queuesToClaimFrom:   []queue.Name{"unclaim-queue-3", "unclaim-queue-4"},
				numberToClaim:       10,
			},

			check: func(result *task.Task, err error) {
				var expected task.NotOwnedByWorker
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "unclaiming a task that belongs to the current worker but has already been marked as complete",
			args: args{
				queuesToSeed:        []queue.Name{"unclaim-queue-5", "unclaim-queue-6"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker1",
				unclaimWithWorker:   "worker1",
				queuesToClaimFrom:   []queue.Name{"unclaim-queue-51", "unclaim-queue-6"},
				numberToClaim:       10,
			},
			beforeUnclaim: func(service *task2.EsService, claimed []task.Task) {
				toClaim := claimed[0]
				_, err := service.MarkDone(ctx, "worker1", toClaim.Queue, toClaim.ID, nil)
				if err != nil {
					t.Error(err)
				}
			},
			check: func(result *task.Task, err error) {
				var expected task.NotClaimed
				assert.IsType(t, expected, err)
			},
		},
		{
			name: "unclaiming a task claimed by the current user",
			args: args{
				queuesToSeed:        []queue.Name{"unclaim-queue-5", "unclaim-queue-6"},
				tasksToSeedPerQueue: 10,
				workerId:            "worker1",
				unclaimWithWorker:   "worker1",
				queuesToClaimFrom:   []queue.Name{"unclaim-queue-51", "unclaim-queue-6"},
				numberToClaim:       10,
			},
			check: func(result *task.Task, err error) {
				assert.Nil(t, err)
				assert.EqualValues(t, task.QUEUED, result.State)
				assert.EqualValues(t, 0, result.Attempted)
				assert.NotNil(t, result.LastClaimed)
			},
		},
	}
	for _, tt := range tests {
		tt := tt // for parallelism
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var seeded []task.Task
			service := buildTasksService()
			for _, queueToSeed := range tt.args.queuesToSeed {
				created := seedTasks(t, service, tt.args.tasksToSeedPerQueue, queueToSeed)
				seeded = append(seeded, created...)
			}

			claimed, err := service.Claim(ctx, tt.args.workerId, tt.args.queuesToClaimFrom, tt.args.numberToClaim, blockFor)
			if err != nil {
				t.Error(err)
			}

			if tt.beforeUnclaim != nil {
				if esService, ok := service.(*task2.EsService); ok {
					tt.beforeUnclaim(esService, claimed)
				}
			}

			if tt.args.forceClaimId != nil && tt.args.forceQueue != nil {
				result, err := service.UnClaim(ctx, tt.args.unclaimWithWorker, *tt.args.forceQueue, *tt.args.forceClaimId)
				tt.check(result, err)
			} else if tt.args.taskFromCreated != nil {
				t := tt.args.taskFromCreated(seeded)
				result, err := service.UnClaim(ctx, tt.args.unclaimWithWorker, t.Queue, t.ID)
				tt.check(result, err)
			} else {
				// Assume we have non-empty claims
				claimedTask := claimed[0]
				result, err := service.UnClaim(ctx, tt.args.unclaimWithWorker, claimedTask.Queue, claimedTask.ID)
				tt.check(result, err)
				if err == nil {
					retrieved, err := service.Get(ctx, claimedTask.Queue, claimedTask.ID)
					if err != nil {
						t.Error(err)
					} else {
						assert.EqualValues(t, task.QUEUED, retrieved.State)
						assert.EqualValues(t, uint(claimedTask.Attempted)-1, uint(retrieved.Attempted))
						assert.NotNil(t, retrieved.LastClaimed)
					}
				}
			}
		})
	}
}

func Test_esTaskService_FailTimedOutTasks(t *testing.T) {
	service := buildTasksService()

	queueToSeed := queue.Name("claimed-tasks-to-expire")
	seededClaimedTasks := seedClaimedTasks(t, service, 100, queueToSeed)
	assert.True(t, len(seededClaimedTasks) > 0)

	// Not yet expired
	if err := service.ReapTimedOutTasks(ctx, 500, 1*time.Minute); err != nil {
		t.Error(err)
	}

	for _, seeded := range seededClaimedTasks {
		retrieved, err := service.Get(ctx, seeded.Queue, seeded.ID)
		if err != nil {
			t.Error(err)
		}
		assert.EqualValues(t, task.CLAIMED, retrieved.State)
	}

	// fast forward time on the time getter
	futureTime := time.Now().Add(seededProcessingTimeout * 5)
	setTasksServiceClock(t, service, futureTime)

	assert.Eventually(t, func() bool {
		seededClaimedTasks := seededClaimedTasks
		if err := service.ReapTimedOutTasks(context.Background(), 500, 1*time.Minute); err != nil {
			return false
		}
		ok := true
		for _, seeded := range seededClaimedTasks {
			retrieved, err := service.Get(context.Background(), seeded.Queue, seeded.ID)
			if err != nil {
				return false
			} else {
				ok = ok && retrieved.State == task.FAILED
			}
		}
		return ok
	}, 120*time.Second, 500*time.Millisecond, "The Tasks we claimed should now be timed out.")

}

var seededProcessingTimeout = 15 * time.Minute

func seedTasks(t *testing.T, service task.Service, numberToSeed int, queue queue.Name) []task.Task {
	runAt := task.RunAt(time.Now().UTC())
	var createdTasks []task.Task

	toCreate := task.NewTask{
		Queue:             queue,
		RetryTimes:        10,
		Priority:          10,
		ProcessingTimeout: task.ProcessingTimeout(seededProcessingTimeout),
		Kind:              task.Kind("justATest"),
		RunAt:             runAt,
		Args:              nil,
		Context:           nil,
	}
	for i := 0; i < numberToSeed; i++ {
		created, err := service.Create(ctx, &toCreate)
		if err != nil {
			t.Error(err)
		}
		createdTasks = append(createdTasks, *created)
	}
	return createdTasks
}

func seedClaimedTasks(t *testing.T, service task.Service, numberToSeed int, queueName queue.Name) []task.Task {
	created := seedTasks(t, service, numberToSeed, queueName)
	var claimed []task.Task
	for len(claimed) != len(created) {
		r, err := service.Claim(ctx, "werk", []queue.Name{queueName}, uint(len(created)), 5*time.Second)
		if err != nil {
			t.Error(err)
		}
		claimed = append(claimed, r...)
	}
	return claimed
}

func setTasksServiceClock(t *testing.T, service task.Service, frozenTime time.Time) {
	// fast forward time on the time getter
	esService, ok := service.(*task2.EsService)
	if !ok {
		t.Error("Not esTaskService")
	}
	// Now expired
	esService.SetUTCGetter(func() time.Time {
		return frozenTime
	})

}
