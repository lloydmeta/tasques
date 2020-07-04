package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/worker"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
)

var TasquesQueuePrefix = ".tasques_queue-"
var TasquesArchiveIndex = ".tasques_archive"

type EsService struct {
	client   *elasticsearch.Client
	settings config.Tasks

	mu sync.Mutex
	// This is an thread-unsafe cache, all access needs to be wrapped via the above mutex
	queueToLastTouchedTimes *lru.Cache

	getUTC func() time.Time // for mocking
}

// For testing
func (e *EsService) SetUTCGetter(getter func() time.Time) {
	e.getUTC = getter
}

func (e *EsService) tasksDefaultsSettings() *config.TasksDefaults {
	return &e.settings.Defaults
}

func (e *EsService) queuesSettings() *config.Queues {
	return &e.settings.Queues
}

func NewService(client *elasticsearch.Client, settings config.Tasks) task.Service {
	queueToLastTouchedTimes := lru.New(int(settings.Queues.LastActivityTrackerMaxSize))
	return &EsService{
		client:                  client,
		settings:                settings,
		mu:                      sync.Mutex{},
		queueToLastTouchedTimes: queueToLastTouchedTimes,
		getUTC: func() time.Time {
			return time.Now().UTC()
		}}
}

func (e *EsService) Create(ctx context.Context, newTask *task.NewTask) (*task.Task, error) {
	indexName := BuildIndexName(newTask.Queue)
	now := e.getUTC()

	taskId := task.GenerateId(newTask)
	toPersist := PersistedTaskData{
		Id:                string(taskId),
		RetryTimes:        uint(newTask.RetryTimes),
		Queue:             string(newTask.Queue),
		RemainingAttempts: uint(newTask.RetryTimes) + 1, // even with 0 retries, we want 1 attempt
		Kind:              string(newTask.Kind),
		State:             task.QUEUED,
		RunAt:             (time.Time)(newTask.RunAt),
		Priority:          int(newTask.Priority),
		Args:              (*jsonObjMap)(newTask.Args),
		ProcessingTimeout: time.Duration(newTask.ProcessingTimeout),
		Context:           (*jsonObjMap)(newTask.Context),
		LastEnqueuedAt:    now,
		LastClaimed:       nil,
		Metadata: common.PersistedMetadata{
			CreatedAt:  now,
			ModifiedAt: now,
		},
		RecurringTaskId: (*string)(newTask.RecurringTaskId),
	}

	toPersistBytes, err := json.Marshal(toPersist)
	if err != nil {
		return nil, common.JsonSerdesErr{Underlying: []error{err}}
	}

	indexReq := esapi.CreateRequest{
		DocumentID: string(taskId),
		Index:      string(indexName),
		Body:       bytes.NewReader(toPersistBytes),
	}

	rawResp, err := indexReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	statusCode := rawResp.StatusCode
	switch {
	case 200 <= statusCode && statusCode <= 299:
		var response common.EsCreateResponse
		if err := json.NewDecoder(rawResp.Body).Decode(&response); err != nil {
			return nil, common.JsonSerdesErr{Underlying: []error{err}}
		}
		domainTask := toPersist.toDomainTask(response.Version())
		return &domainTask, nil
	case statusCode == 409:
		return nil, task.AlreadyExists{ID: taskId}
	default:
		return nil, common.UnexpectedEsStatusError(rawResp)
	}

}

func (e *EsService) Get(ctx context.Context, queue queue.Name, taskId task.Id) (*task.Task, error) {
	searchReq := esapi.GetRequest{
		Index:      string(BuildIndexName(queue)),
		DocumentID: string(taskId),
	}
	rawResp, err := searchReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()

	switch rawResp.StatusCode {
	case 200:
		var response esHitPersistedTask
		if err := json.NewDecoder(rawResp.Body).Decode(&response); err != nil {
			return nil, common.JsonSerdesErr{Underlying: []error{err}}
		}
		retrieved := response.ToDomainTask()
		return &retrieved, nil
	case 404:
		return nil, task.NotFound{ID: taskId, QueueName: queue}
	default:
		return nil, common.UnexpectedEsStatusError(rawResp)
	}
}

func (e *EsService) Claim(ctx context.Context, workerId worker.Id, queues []queue.Name, desiredTasks uint, blockFor time.Duration) ([]task.Task, error) {
	var allClaimed []task.Task

	startUTC := e.getUTC()

	retryWait := e.retryWait(blockFor)

	firstTry := true

	for uint(len(allClaimed)) < desiredTasks && e.getUTC().Sub(startUTC) < blockFor {
		if !firstTry {
			time.Sleep(retryWait)
		}
		nextDesiredCount := desiredTasks - uint(len(allClaimed))
		if claimed, err := e.searchAndClaim(ctx, workerId, queues, nextDesiredCount, startUTC, blockFor, retryWait); err != nil {
			return nil, err
		} else {
			allClaimed = append(allClaimed, claimed...)
			firstTry = false
		}
	}
	return allClaimed, nil
}

// retryWait returns the amount of time to wait between retries, based on settings *and* a given block for duration
//
// It does this by first defaulting to the min claim retry wait, but if claim retry max retries count is set greater than 0,
// calculates the even wait between retries by dividing blockFor by max retries count, and picks the longer duration (
// between that and min claim retry wait.
//
// In all cases, the min claim retry wait is respected, so as to not flood the ES server with requests.
func (e *EsService) retryWait(reqBlockFor time.Duration) time.Duration {
	retryWait := e.tasksDefaultsSettings().BlockForRetryMinWait
	if e.tasksDefaultsSettings().BlockForRetryMaxRetries > 0 {
		evenRetryWait := time.Duration(uint64(reqBlockFor.Nanoseconds())/uint64(e.tasksDefaultsSettings().BlockForRetryMaxRetries)) * time.Nanosecond
		if evenRetryWait > retryWait {
			retryWait = evenRetryWait
		}
	}
	return retryWait
}

func (e *EsService) ReportIn(ctx context.Context, workerId worker.Id, queue queue.Name, taskId task.Id, newReport task.NewReport) (*task.Task, error) {
	reportedInAt := e.getUTC()

	runUpdate := func() (*task.Task, error) {
		return e.getAndUpdate(ctx, queue, taskId, func(targetTask *task.Task) error {
			return targetTask.ReportIn(workerId, newReport, task.ReportedAt(reportedInAt))
		}, metadata.ModifiedAt(reportedInAt))
	}
	result, err := runUpdate()
	timesRetried := uint(0)
	if _, isVersionConflict := err.(task.InvalidVersion); isVersionConflict && timesRetried < e.tasksDefaultsSettings().VersionConflictRetryTimes {
		timesRetried++
		result, err = runUpdate()
	}
	return result, err
}

func (e *EsService) MarkDone(ctx context.Context, workerId worker.Id, queue queue.Name, taskId task.Id, success *task.Success) (*task.Task, error) {
	return e.markComplete(ctx, queue, taskId, func(targetTask *task.Task, at task.CompletedAt) error {
		return targetTask.IntoDone(workerId, at, success)
	})
}

func (e *EsService) MarkFailed(ctx context.Context, workerId worker.Id, queue queue.Name, taskId task.Id, failure *task.Failure) (*task.Task, error) {
	return e.markComplete(ctx, queue, taskId, func(targetTask *task.Task, at task.CompletedAt) error {
		return targetTask.IntoFailed(workerId, at, failure)
	})
}

func (e *EsService) UnClaim(ctx context.Context, workerId worker.Id, queue queue.Name, taskId task.Id) (*task.Task, error) {
	runUpdate := func() (*task.Task, error) {
		unclaimedAt := e.getUTC() // always grab a later date for unclaiming
		return e.getAndUpdate(ctx, queue, taskId, func(targetTask *task.Task) error {
			return targetTask.IntoUnClaimed(workerId, task.EnqueuedAt(unclaimedAt))
		}, metadata.ModifiedAt(unclaimedAt))
	}
	result, err := runUpdate()
	timesRetried := uint(0)
	if _, isVersionConflict := err.(task.InvalidVersion); isVersionConflict && timesRetried < e.tasksDefaultsSettings().VersionConflictRetryTimes {
		timesRetried++
		result, err = runUpdate()
	}
	return result, err
}

func (e *EsService) ReapTimedOutTasks(ctx context.Context, scrollSize uint, scrollTtl time.Duration) error {
	now := e.getUTC()
	searchBody := buildTimedOutSearchBody(now, scrollSize)
	return e.scanTasks(ctx, searchBody, scrollTtl, func(timedOutTasks []task.Task) error {

		log.Info().Int("timed_out_tasks_count", len(timedOutTasks)).Msg("Timed out tasks in batch")
		aboutToTimeOut := make([]task.Task, 0, len(timedOutTasks))
		for _, t := range timedOutTasks {
			// shouldn't happen but eh
			if t.LastClaimed != nil {
				// also shouldn't happen ...
				if failErr := t.IntoFailed(t.LastClaimed.WorkerId, task.CompletedAt(now), &task.Failure{"timed_out": "Did not finish before claim timeout period"}); failErr == nil {
					t.Metadata.ModifiedAt = metadata.ModifiedAt(now)
					aboutToTimeOut = append(aboutToTimeOut, t)
				}
			}
		}

		log.Info().Msg("Sending bulk request to time out the batch")
		if _, err := e.bulkUpdateTasks(ctx, aboutToTimeOut); err != nil {
			return err
		} else {
			log.Info().Msg("Bulk timeout request sent and acked")
			return nil
		}
	})
}

func (e *EsService) ArchiveOldTasks(ctx context.Context, archiveCompletedBefore task.CompletedAt, scrollSize uint, scrollTtl time.Duration) error {
	if log.Debug().Enabled() {
		log.Debug().
			Time("archive_completed_before", time.Time(archiveCompletedBefore)).
			Msg("Looking for tasks completed before")
	}
	searchBody := buildArchivableSearchBody(time.Time(archiveCompletedBefore), scrollSize)
	return e.scanTasks(ctx, searchBody, scrollTtl, func(archivableTasks []task.Task) error {
		if log.Debug().Enabled() {
			log.Debug().
				Interface("archivable_tasks", archivableTasks).
				Msg("Archivable Tasks in batch")
		}
		log.Info().Int("archivable_tasks_count", len(archivableTasks)).Msg("Archivable Tasks in batch")
		log.Info().Msg("Sending bulk request to archive the batch")
		if _, err := e.bulkArchiveTasks(ctx, archivableTasks); err != nil {
			return err
		} else {
			log.Info().Msg("Bulk timeout request sent and acked")
			return nil
		}
	})
}

// Scrolls through all tasks using a search body, taking care to close all response bodies and close scrolls
func (e *EsService) scanTasks(ctx context.Context, searchBody jsonObjMap, scrollTtl time.Duration, doWithBatch func(tasks []task.Task) error) error {
	log.Debug().Msg("Beginning to scan tasks .. ")
	log.Debug().Interface("searchBody", searchBody).Msg("Scanning tasks")
	tasksWithScrollId, err := e.initSearch(ctx, searchBody, scrollTtl)
	if err != nil {
		return err
	}
	scannedTasks := tasksWithScrollId.Tasks
	var scrollIds []string
	scrollId := tasksWithScrollId.ScrollId
	scrollIds = append(scrollIds, scrollId)
	defer func() {
		if scrollErr := e.clearScroll(ctx, scrollIds); scrollErr != nil && err == nil {
			err = scrollErr
		}
	}()

	for len(scannedTasks) > 0 {
		if err := doWithBatch(scannedTasks); err != nil {
			return err
		}
		nextTasksWithScrollId, err := e.scroll(ctx, scrollId, scrollTtl)
		if err != nil {
			return err
		}
		scannedTasks = nextTasksWithScrollId.Tasks
		scrollId = nextTasksWithScrollId.ScrollId
		scrollIds = append(scrollIds, nextTasksWithScrollId.ScrollId)
	}
	log.Debug().Msg("Scanning tasks end ")
	return nil
}

func (e *EsService) initSearch(ctx context.Context, searchBody jsonObjMap, scrollTtl time.Duration) (*tasksWithScrollId, error) {
	searchBodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, common.JsonSerdesErr{Underlying: []error{err}}
	}
	searchReq := esapi.SearchRequest{
		Scroll:         scrollTtl, // make this configurable
		Index:          []string{string(allTaskqueueQueuesPattern)},
		AllowNoIndices: esapi.BoolPtr(true),
		Body:           bytes.NewReader(searchBodyBytes),
	}

	rawResp, err := searchReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	return processScrollResp(rawResp)
}

func (e *EsService) scroll(ctx context.Context, scrollId string, scrollTtl time.Duration) (*tasksWithScrollId, error) {

	scrollReq := esapi.ScrollRequest{
		Scroll:   scrollTtl,
		ScrollID: scrollId,
	}

	rawResp, err := scrollReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	return processScrollResp(rawResp)
}

func processScrollResp(rawResp *esapi.Response) (*tasksWithScrollId, error) {
	switch rawResp.StatusCode {
	case 200:
		var scrollResp esSearchScrollingResponse
		if err := json.NewDecoder(rawResp.Body).Decode(&scrollResp); err != nil {
			return nil, common.JsonSerdesErr{Underlying: []error{err}}
		}
		tasks := make([]task.Task, 0, len(scrollResp.Hits.Hits))
		for _, pTask := range scrollResp.Hits.Hits {
			tasks = append(tasks, pTask.ToDomainTask())
		}
		return &tasksWithScrollId{
			ScrollId: scrollResp.ScrollId,
			Tasks:    tasks,
		}, nil
	default:
		return nil, common.UnexpectedEsStatusError(rawResp)
	}
}

func (e *EsService) RefreshAsNeeded(ctx context.Context, name queue.Name) error {
	// Note don't do locking in this method; we do it in the methods that actually read from the
	// needs refreshing map. This keeps the locking _short_ so that we can mark touched
	// from other methods quickly. The actual avoidance of refreshing is a best effort optimisation anyways
	// currently exclusively for Recurring Tasks. so it's not worth it to to make a huge effort
	// there
	if e.needsRefreshing(name) {
		refreshReq := esapi.IndicesRefreshRequest{
			Index:             []string{string(name)},
			AllowNoIndices:    esapi.BoolPtr(true),
			IgnoreUnavailable: esapi.BoolPtr(true),
		}
		rawResp, err := refreshReq.Do(ctx, e.client)
		if err != nil {
			return common.ElasticsearchErr{Underlying: err}
		}
		defer rawResp.Body.Close()
		respStatus := rawResp.StatusCode
		switch {
		case 200 <= respStatus && respStatus <= 299:
			e.markTouched(name)
			return nil
		default:
			return common.UnexpectedEsStatusError(rawResp)
		}
	} else {
		return nil
	}
}

func (e *EsService) OutstandingTasksCount(ctx context.Context, queue queue.Name, recurringTaskId task.RecurringTaskId) (uint, error) {
	searchBody := buildOutstandingTasksCountQuery(recurringTaskId)
	searchBodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return 0, common.JsonSerdesErr{Underlying: []error{err}}
	}

	countRequest := esapi.CountRequest{
		Index:             []string{string(BuildIndexName(queue))},
		Body:              bytes.NewReader(searchBodyBytes),
		AllowNoIndices:    esapi.BoolPtr(true),
		IgnoreUnavailable: esapi.BoolPtr(true),
	}
	rawResp, err := countRequest.Do(ctx, e.client)
	if err != nil {
		return 0, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	switch rawResp.StatusCode {
	case 200:
		var countResp common.EsCountResponse
		if err := json.NewDecoder(rawResp.Body).Decode(&countResp); err != nil {
			return 0, common.JsonSerdesErr{Underlying: []error{err}}
		}
		return countResp.Count, nil
	default:
		return 0, common.UnexpectedEsStatusError(rawResp)
	}
}

func (e *EsService) markTouched(name queue.Name) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Explicit type ascription because we need to cast below...thanks Golang for not having generics
	var now time.Time = e.getUTC()
	e.queueToLastTouchedTimes.Add(name, now)
}

func (e *EsService) markAllTouched(names []queue.Name) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Explicit type ascription because we need to cast below...thanks Golang for not having generics
	var now time.Time = e.getUTC()
	for _, name := range names {
		e.queueToLastTouchedTimes.Add(name, now)
	}
}

func (e *EsService) needsRefreshing(name queue.Name) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if lastTouchedInterface, ok := e.queueToLastTouchedTimes.Get(name); ok {
		// The nasty cast. Thanks for "keeping it simple", Golang and not supporting Generics
		// /sarcasm.
		if lastTouched, ok := lastTouchedInterface.(time.Time); ok {
			now := e.getUTC()
			diff := now.Sub(lastTouched)
			needsRefresh := diff > e.queuesSettings().RefreshIfLastTouchedOver
			if log.Debug().Enabled() {
				log.Debug().
					Str("queue", string(name)).
					Time("now", now).
					Time("lastTouched", lastTouched).
					Dur("shouldRefreshIfDiffBiggerThan", e.queuesSettings().RefreshIfLastTouchedOver).
					Dur("diff", diff).
					Bool("needsRefresh", needsRefresh).
					Msg("Check if queue needs refreshing")
			}
			return needsRefresh
		} else {
			log.Error().Msg("Somehow got a class cast error accessing an internal untyped map. FML.")
			return true
		}
	} else {
		if log.Debug().Enabled() {
			log.Debug().
				Str("queue", string(name)).
				Msg("No entry in queueToLastTouchedTimes, returning true")
		}
		return true
	}
}

// Helper to mark a task as completed (success or failed)
func (e *EsService) markComplete(ctx context.Context, queue queue.Name, taskId task.Id, mutate func(task *task.Task, at task.CompletedAt) error) (*task.Task, error) {
	runUpdate := func() (*task.Task, error) {
		// unlike reporting in, when completing, we always try to complete it with a newer date because it
		// should be the last thing that happened
		completedAt := e.getUTC()
		return e.getAndUpdate(ctx, queue, taskId, func(targetTask *task.Task) error {
			return mutate(targetTask, task.CompletedAt(completedAt))
		}, metadata.ModifiedAt(completedAt))
	}
	result, err := runUpdate()
	timesRetried := uint(0)
	if _, isVersionConflict := err.(task.InvalidVersion); isVersionConflict && timesRetried < e.tasksDefaultsSettings().VersionConflictRetryTimes {
		timesRetried++
		result, err = runUpdate()
	}
	return result, err
}

func (e *EsService) getAndUpdate(ctx context.Context, queue queue.Name, taskId task.Id, mutate func(targetTask *task.Task) error, at metadata.ModifiedAt) (*task.Task, error) {
	targetTask, err := e.Get(ctx, queue, taskId)

	// Error checks
	if err != nil {
		return nil, err
	}
	if err := mutate(targetTask); err != nil {
		return nil, err
	}

	targetTask.Metadata.ModifiedAt = at

	// Build mutate data
	updatePayload := ToPersistedTask(targetTask)
	updatePayloadBytes, err := json.Marshal(updatePayload)
	if err != nil {
		return nil, common.JsonSerdesErr{Underlying: []error{err}}
	}
	// Purposely using the Index API (rather than the update API) so as to
	// not get bit by old stale data due to partial updates. We send optimistic
	// locking data to ensure we are _updating_
	updateReq := esapi.IndexRequest{
		Index:         string(BuildIndexName(queue)),
		DocumentID:    string(taskId),
		Body:          bytes.NewReader(updatePayloadBytes),
		IfPrimaryTerm: esapi.IntPtr(int(targetTask.Metadata.Version.PrimaryTerm)),
		IfSeqNo:       esapi.IntPtr(int(targetTask.Metadata.Version.SeqNum)),
	}
	rawResp, err := updateReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	respStatus := rawResp.StatusCode
	switch {
	case 200 <= respStatus && respStatus <= 299:
		// Updated, grab new metadata
		var resp common.EsUpdateResponse
		if err := json.NewDecoder(rawResp.Body).Decode(&resp); err != nil {
			return nil, common.JsonSerdesErr{Underlying: []error{err}}
		}
		targetTask.Metadata.Version = resp.Version()
		return targetTask, nil
	case respStatus == 409:
		return nil, task.InvalidVersion{ID: taskId}
	case respStatus == 404:
		return nil, task.NotFound{
			ID:        taskId,
			QueueName: queue,
		}
	default:
		return nil, common.UnexpectedEsStatusError(rawResp)
	}

}

type jsonObjMap map[string]interface{}

func BuildIndexName(queue queue.Name) common.IndexName {
	return common.IndexName(fmt.Sprintf("%s%s", TasquesQueuePrefix, string(queue)))
}

var allTaskqueueQueuesPattern = BuildIndexName("*")

func queueNameFromIndexName(indexName common.IndexName) queue.Name {
	return queue.Name(strings.TrimPrefix(string(indexName), TasquesQueuePrefix))
}

func (e *EsService) searchAndClaim(ctx context.Context, workerId worker.Id, queues []queue.Name, desiredTasks uint, startedAtUTC time.Time, blockFor time.Duration, retryLoopWait time.Duration) ([]task.Task, error) {
	var claimed []task.Task

	// Search for unclaimed Tasks
	searchLimit := e.tasksDefaultsSettings().ClaimAmountSearchMultiplier * desiredTasks
	var searchResults []task.Task
	var err error
	firstTry := true
	// keep looping if we don't have any results at all and haven't blown past our blockFor, but making sure to sleep
	// in between so we don't flood ES
	for len(searchResults) == 0 && e.getUTC().Sub(startedAtUTC) < blockFor {
		if !firstTry {
			time.Sleep(retryLoopWait)
		}
		searchResults, err = e.searchForClaimables(ctx, queues, searchLimit)
		if err != nil {
			return nil, err
		}
		firstTry = false
	}
	e.markAllTouched(queues)

	/*
	 * Try to claim the number of Tasks we want from the Search response, and keep looping through a window until we
	 * either have the number of tasks we need, or we've run out of Tasks to try in the response.
	 *
	 * We do this so that we can decrease the number of search requests to ES _and_ increase the chances of quickly
	 * getting the number of Claims we want.
	 *
	 * Given searchResults [1, 2, 3, 4, 5] (searchResults len is 5)
	 * - Assuming we want to claim 4 in total.
	 * - Starting from zero claimed
	 * To start:
	 * 	1. attemptClaimStartIdx = len(claimed) = 0
	 *  2. targetClaimsLength = 4 - len(claimed) = 4
	 *  3. attemptClaimEndIdx = min(0+4, 5) = 4
	 *  4. aboutToClaims = searchResults[0:4] = [1,2,3,4]
	 *  5. attemptClaimStartIdx = attemptClaimStartIdx+4 = 0 + 4
	 * Assuming we can only claim 2, 4 on attempt 1 of makeClaim; claimed becomes [2, 4]
	 *  1. targetClaimsLength = 4 - len(claimed) = 2
	 *  2. attemptClaimEndIdx = min(attemptClaimStartIdx+2, 5) = min(4+2, 5) = 5
	 *  3. aboutToClaims = searchResults[4:5] = [5]
	 *  4. Assuming we succeed in claiming 5, claimed becomes [2,4,5], we are still lacking 1
	 *  5. attemptClaimStartIdx = attemptClaimStartIdx+1 = 4 + 1 = 5
	 * We exit the loop here attemptClaimStartIdx == len(searchResults)
	 */
	attemptClaimStartIdx := len(claimed)
	for uint(len(claimed)) < desiredTasks && attemptClaimStartIdx < len(searchResults) {
		targetClaimsLength := int(desiredTasks) - len(claimed)
		attemptClaimEndIdx := minInt(attemptClaimStartIdx+targetClaimsLength, len(searchResults))
		aboutToClaims := searchResults[attemptClaimStartIdx:attemptClaimEndIdx]
		if claimedInAttempt, err := e.makeClaim(ctx, workerId, aboutToClaims); err != nil {
			return nil, err
		} else {
			claimed = append(claimed, claimedInAttempt...)
		}
		attemptClaimStartIdx = attemptClaimStartIdx + targetClaimsLength
	}

	return claimed, nil

}

func (e *EsService) searchForClaimables(ctx context.Context, queues []queue.Name, retrieveLimit uint) ([]task.Task, error) {
	indices := make([]string, 0, len(queues))
	if len(queues) == 0 {
		indices = append(indices, string(allTaskqueueQueuesPattern))
	} else {
		for _, q := range queues {
			indices = append(indices, string(BuildIndexName(q)))
		}
	}

	nowUTC := e.getUTC()
	queryBody := buildClaimableQueryBody(retrieveLimit, nowUTC)
	queryBodyAsJsonBytes, err := json.Marshal(queryBody)

	if err != nil {
		return nil, common.JsonSerdesErr{Underlying: []error{err}}
	}

	searchReq := esapi.SearchRequest{
		Index:             indices,
		IgnoreUnavailable: esapi.BoolPtr(true),
		AllowNoIndices:    esapi.BoolPtr(true),
		Body:              bytes.NewReader(queryBodyAsJsonBytes),
	}

	rawResp, err := searchReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()

	switch rawResp.StatusCode {
	case 200:
		var searchResp esSearchResponse
		if err := json.NewDecoder(rawResp.Body).Decode(&searchResp); err != nil {
			return nil, common.JsonSerdesErr{Underlying: []error{err}}
		}
		tasks := make([]task.Task, 0, len(searchResp.Hits.Hits))
		for _, pTask := range searchResp.Hits.Hits {
			tasks = append(tasks, pTask.ToDomainTask())
		}
		return tasks, nil
	case 404:
		return nil, nil
	default:
		return nil, common.UnexpectedEsStatusError(rawResp)
	}
}

func (e *EsService) makeClaim(ctx context.Context, workerId worker.Id, aboutToClaims []task.Task) ([]task.Task, error) {
	var claimed []task.Task

	// Build a BulkRequest to Claim the amount of Tasks that we need.
	claimedAt := task.ClaimedAt(e.getUTC())
	// holy crap `range` returns a copy`
	for i := 0; i < len(aboutToClaims); i++ {
		toClaim := &aboutToClaims[i]
		if err := toClaim.IntoClaimed(workerId, claimedAt); err != nil {
			return nil, err
		}
		toClaim.Metadata.ModifiedAt = metadata.ModifiedAt(time.Time(claimedAt))
	}
	bulkReqResponse, err := e.bulkUpdateTasks(ctx, aboutToClaims)
	if err != nil {
		return nil, err
	}

	// Find how many we were able to claim, ignore the ones that had errors
	for idx, attemptToClaim := range aboutToClaims {
		// we are guaranteed to get the the responses in the same order that the bulk request was built
		bulkResultInfoForClaim := bulkReqResponse.Items[idx].Info()
		if bulkResultInfoForClaim.IsOk() {
			// Claimed! so grab the updated version and append to result list
			attemptToClaim.Metadata.Version = metadata.Version{
				SeqNum:      metadata.SeqNum(bulkResultInfoForClaim.SeqNum),
				PrimaryTerm: metadata.PrimaryTerm(bulkResultInfoForClaim.PrimaryTerm),
			}
			claimed = append(claimed, attemptToClaim)
		}
		// otherwise claimed by someone else, _or_ missing, or there was an error. Either way, safe to ignore.
	}

	return claimed, nil
}

func (e *EsService) bulkUpdateTasks(ctx context.Context, tasks []task.Task) (*common.EsBulkResponse, error) {
	bulkReqBody, err := buildTasksBulkUpdateNdJsonBytes(tasks)
	if err != nil {
		return nil, err
	} else {
		return e.sendBulk(ctx, bulkReqBody)
	}
}

func (e *EsService) bulkArchiveTasks(ctx context.Context, tasks []task.Task) (*common.EsBulkResponse, error) {
	bulkReqBody, err := buildTasksBulkArchiveNdJsonBytes(tasks)
	if err != nil {
		return nil, err
	} else {
		return e.sendBulk(ctx, bulkReqBody)
	}
}

func (e *EsService) sendBulk(ctx context.Context, bulkReqBody []byte) (*common.EsBulkResponse, error) {
	claimBulkReq := esapi.BulkRequest{
		Body: bytes.NewReader(bulkReqBody),
	}
	rawResp, err := claimBulkReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	if rawResp.IsError() {
		return nil, common.UnexpectedEsStatusError(rawResp)
	}
	var response common.EsBulkResponse
	if err := json.NewDecoder(rawResp.Body).Decode(&response); err != nil {
		return nil, common.JsonSerdesErr{Underlying: []error{err}}
	}
	return &response, nil
}

func (e *EsService) clearScroll(ctx context.Context, scrollIds []string) error {
	if len(scrollIds) > 0 {
		clearScrollReq := esapi.ClearScrollRequest{ScrollID: scrollIds}
		rawResp, err := clearScrollReq.Do(ctx, e.client)
		if err != nil {
			return err
		} else {
			defer rawResp.Body.Close()
			switch rawResp.StatusCode {
			case 200:
				return nil
			default:
				return common.UnexpectedEsStatusError(rawResp)
			}
		}
	} else {
		return nil
	}
}

func minInt(x, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

func buildTasksBulkUpdateNdJsonBytes(task []task.Task) ([]byte, error) {
	var errAcc []error
	var bytesAcc []byte
	for _, t := range task {
		pair := buildUpdateBulkOp(&t)
		opBytes, err := json.Marshal(pair.op)
		if err != nil {
			errAcc = append(errAcc, err)
		}
		if len(errAcc) == 0 {
			bytesAcc = append(bytesAcc, opBytes...)
			bytesAcc = append(bytesAcc, "\n"...)
		}

		dataBytes, err := json.Marshal(pair.doc)
		if err != nil {
			errAcc = append(errAcc, err)
		}
		if len(errAcc) == 0 {
			bytesAcc = append(bytesAcc, dataBytes...)
			bytesAcc = append(bytesAcc, "\n"...)
		}
	}
	if len(errAcc) != 0 {
		return nil, common.JsonSerdesErr{Underlying: errAcc}
	} else {
		return bytesAcc, nil
	}
}

func buildTasksBulkArchiveNdJsonBytes(task []task.Task) ([]byte, error) {
	var errAcc []error
	var bytesAcc []byte
	for _, t := range task {
		archival := buildTaskArchival(&t)
		deleteOpBytes, err := json.Marshal(archival.deleteOp)
		if err != nil {
			errAcc = append(errAcc, err)
		}
		if len(errAcc) == 0 {
			bytesAcc = append(bytesAcc, deleteOpBytes...)
			bytesAcc = append(bytesAcc, "\n"...)
		}
		archiveOpBytes, err := json.Marshal(archival.insertArchiveTaskOp.op)
		if err != nil {
			errAcc = append(errAcc, err)
		}
		if len(errAcc) == 0 {
			bytesAcc = append(bytesAcc, archiveOpBytes...)
			bytesAcc = append(bytesAcc, "\n"...)
		}
		dataBytes, err := json.Marshal(archival.insertArchiveTaskOp.doc)
		if err != nil {
			errAcc = append(errAcc, err)
		}
		if len(errAcc) == 0 {
			bytesAcc = append(bytesAcc, dataBytes...)
			bytesAcc = append(bytesAcc, "\n"...)
		}
	}
	if len(errAcc) != 0 {
		return nil, common.JsonSerdesErr{Underlying: errAcc}
	} else {
		return bytesAcc, nil
	}
}

func buildUpdateBulkOp(task *task.Task) updateTaskBulkOpPair {
	return updateTaskBulkOpPair{
		op: updateTaskBulkPairOp{
			Index: updateTaskBulkPairOpData{
				Id:            string(task.ID),
				Index:         string(BuildIndexName(task.Queue)),
				IfSeqNo:       uint64(task.Metadata.Version.SeqNum),
				IfPrimaryTerm: uint64(task.Metadata.Version.PrimaryTerm),
			},
		},
		doc: ToPersistedTask(task),
	}
}

func buildTaskArchival(t *task.Task) bulkTaskArchival {
	return bulkTaskArchival{
		deleteOp: deleteBulkOp{
			Delete: deleteBulkOpData{
				Id:            string(t.ID),
				Index:         string(BuildIndexName(t.Queue)),
				IfSeqNo:       uint64(t.Metadata.Version.SeqNum),
				IfPrimaryTerm: uint64(t.Metadata.Version.PrimaryTerm),
			},
		},
		insertArchiveTaskOp: insertArchivedTaskOpPair{
			op: insertArchivedTaskOp{
				Index: insertArchivedTaskOpData{
					Index: TasquesArchiveIndex,
				},
			},
			doc: ToPersistedTask(t),
		},
	}
}

// Private persistence doc structures based entirely on basic types for ease of guaranteeing serdes.

type PersistedTaskData struct {
	Id         string `json:"id"`
	Queue      string `json:"queue"`
	RetryTimes uint   `json:"retry_times"`
	// This doesn't map to the domain model 1:1 because storing _attempts_ instead of retires
	// allows us to not need to adjust the counts for timeouts, and instead issue a simple update-by-query
	RemainingAttempts uint                     `json:"remaining_attempts"`
	Kind              string                   `json:"kind"`
	State             task.State               `json:"state"`
	RunAt             time.Time                `json:"run_at"`
	ProcessingTimeout time.Duration            `json:"processing_timeout"`
	Priority          int                      `json:"priority"`
	Args              *jsonObjMap              `json:"args,omitempty"`
	Context           *jsonObjMap              `json:"context,omitempty"`
	LastEnqueuedAt    time.Time                `json:"last_enqueued_at"`
	LastClaimed       *persistedLastClaimed    `json:"last_claimed,omitempty"`
	Metadata          common.PersistedMetadata `json:"metadata"`
	RecurringTaskId   *string                  `json:"recurring_task_id,omitempty"`
}

type persistedReport struct {
	At   time.Time   `json:"at"`
	Data *jsonObjMap `json:"data,omitempty"`
}

type persistedLastClaimed struct {
	WorkerId   string           `json:"worker_id"`
	ClaimedAt  time.Time        `json:"claimed_at"`
	TimesOutAt time.Time        `json:"times_out_at"`
	LastReport *persistedReport `json:"last_report,omitempty"`
	Result     *persistedResult `json:"result,omitempty"`
}

type persistedResult struct {
	At time.Time `json:"at"`
	// Results. Only one of the following will be filled in at a given time
	Failure *jsonObjMap `json:"failure,omitempty"`
	Success *jsonObjMap `json:"success,omitempty"`
}

func (pTask *PersistedTaskData) toDomainTask(version metadata.Version) task.Task {
	var domainLastClaimed *task.LastClaimed
	if pTask.LastClaimed != nil {
		var result *task.Result
		if pTask.LastClaimed.Result != nil {
			result = &task.Result{
				At:      task.CompletedAt(pTask.LastClaimed.Result.At),
				Failure: (*task.Failure)(pTask.LastClaimed.Result.Failure),
				Success: (*task.Success)(pTask.LastClaimed.Result.Success),
			}
		}
		var lastReport *task.Report
		if pTask.LastClaimed.LastReport != nil {
			lastReport = &task.Report{
				At:   task.ReportedAt(pTask.LastClaimed.LastReport.At),
				Data: (*task.ReportedData)(pTask.LastClaimed.LastReport.Data),
			}
		}
		domainLastClaimed = &task.LastClaimed{
			WorkerId:   worker.Id(pTask.LastClaimed.WorkerId),
			ClaimedAt:  task.ClaimedAt(pTask.LastClaimed.ClaimedAt),
			TimesOutAt: task.TimesOutAt(pTask.LastClaimed.TimesOutAt),
			LastReport: lastReport,
			Result:     result,
		}
	}

	return task.Task{
		ID:                task.Id(pTask.Id),
		Queue:             queue.Name(pTask.Queue),
		RetryTimes:        task.RetryTimes(pTask.RetryTimes),
		Attempted:         task.AttemptedTimes(pTask.RetryTimes + 1 - pTask.RemainingAttempts),
		Kind:              task.Kind(pTask.Kind),
		State:             pTask.State,
		Priority:          task.Priority(pTask.Priority),
		RunAt:             task.RunAt(pTask.RunAt),
		Args:              (*task.Args)(pTask.Args),
		ProcessingTimeout: task.ProcessingTimeout(pTask.ProcessingTimeout),
		Context:           (*task.Context)(pTask.Context),
		LastClaimed:       domainLastClaimed,
		LastEnqueuedAt:    task.EnqueuedAt(pTask.LastEnqueuedAt),
		Metadata: metadata.Metadata{
			CreatedAt:  metadata.CreatedAt(pTask.Metadata.CreatedAt),
			ModifiedAt: metadata.ModifiedAt(pTask.Metadata.ModifiedAt),
			Version:    version,
		},
		RecurringTaskId: (*task.RecurringTaskId)(pTask.RecurringTaskId),
	}
}

func (resp *esHitPersistedTask) ToDomainTask() task.Task {
	pTask := resp.Source

	return pTask.toDomainTask(metadata.Version{
		SeqNum:      metadata.SeqNum(resp.SeqNum),
		PrimaryTerm: metadata.PrimaryTerm(resp.PrimaryTerm),
	})
}

func ToPersistedTask(task *task.Task) PersistedTaskData {
	var lastClaimed *persistedLastClaimed
	if task.LastClaimed != nil {
		var result *persistedResult
		if task.LastClaimed.Result != nil {
			result = &persistedResult{
				At:      time.Time(task.LastClaimed.Result.At),
				Failure: (*jsonObjMap)(task.LastClaimed.Result.Failure),
				Success: (*jsonObjMap)(task.LastClaimed.Result.Success),
			}
		}

		var lastReport *persistedReport
		if task.LastClaimed.LastReport != nil {
			lastReport = &persistedReport{
				At:   time.Time(task.LastClaimed.LastReport.At),
				Data: (*jsonObjMap)(task.LastClaimed.LastReport.Data),
			}
		}

		lastClaimed = &persistedLastClaimed{
			WorkerId:   string(task.LastClaimed.WorkerId),
			ClaimedAt:  time.Time(task.LastClaimed.ClaimedAt),
			TimesOutAt: time.Time(task.LastClaimed.TimesOutAt),
			LastReport: lastReport,
			Result:     result,
		}
	}

	return PersistedTaskData{
		Id:                string(task.ID),
		RetryTimes:        uint(task.RetryTimes),
		Queue:             string(task.Queue),
		RemainingAttempts: uint(task.RetryTimes) + 1 - uint(task.Attempted),
		Kind:              string(task.Kind),
		State:             task.State,
		RunAt:             time.Time(task.RunAt),
		Priority:          int(task.Priority),
		Args:              (*jsonObjMap)(task.Args),
		ProcessingTimeout: time.Duration(task.ProcessingTimeout),
		Context:           (*jsonObjMap)(task.Context),
		LastEnqueuedAt:    time.Time(task.LastEnqueuedAt),
		LastClaimed:       lastClaimed,
		Metadata: common.PersistedMetadata{
			CreatedAt:  time.Time(task.Metadata.CreatedAt),
			ModifiedAt: time.Time(task.Metadata.ModifiedAt),
		},
		RecurringTaskId: (*string)(task.RecurringTaskId),
	}
}

type esHitPersistedTask struct {
	ID          string            `json:"_id"`
	Index       string            `json:"_index"`
	SeqNum      uint64            `json:"_seq_no"`
	PrimaryTerm uint64            `json:"_primary_term"`
	Source      PersistedTaskData `json:"_source"`
}

func buildClaimableQueryBody(limit uint, nowUtc time.Time) jsonObjMap {
	return jsonObjMap{
		"from":                0,
		"size":                limit,
		"seq_no_primary_term": true,
		"sort": []jsonObjMap{
			{
				"priority": jsonObjMap{
					"order": "desc",
				},
			},
			{
				"run_at": jsonObjMap{
					"order": "asc",
				},
			},
			{
				"remaining_attempts": jsonObjMap{
					"order": "desc",
				},
			},
		},
		"query": jsonObjMap{
			"bool": jsonObjMap{
				"filter": jsonObjMap{
					"bool": jsonObjMap{
						"must": []jsonObjMap{
							// DO NOT add a requirement that LastClaimed is empty without updating UnClaim
							{
								"range": jsonObjMap{
									"run_at": jsonObjMap{
										"lte": nowUtc.Format(time.RFC3339Nano),
									},
								},
							},
							{
								"range": jsonObjMap{
									"remaining_attempts": jsonObjMap{
										"gt": 0,
									},
								},
							}, {
								"bool": jsonObjMap{
									"should": []jsonObjMap{
										{
											"term": jsonObjMap{
												"state": task.QUEUED.String(), // just queued
											},
										},
										{
											"term": jsonObjMap{
												"state": task.FAILED.String(), // we can retry
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

type esSearchResponse struct {
	Hits struct {
		Hits []esHitPersistedTask `json:"hits"`
	} `json:"hits"`
}

type updateTaskBulkOpPair struct {
	op  updateTaskBulkPairOp
	doc PersistedTaskData
}

type updateTaskBulkPairOp struct {
	Index updateTaskBulkPairOpData `json:"index"`
}

type updateTaskBulkPairOpData struct {
	Id      string `json:"_id"`
	Index   string `json:"_index"`
	IfSeqNo uint64 `json:"if_seq_no"`

	IfPrimaryTerm uint64 `json:"if_primary_term"`
}

type bulkTaskArchival struct {
	deleteOp            deleteBulkOp
	insertArchiveTaskOp insertArchivedTaskOpPair
}

type deleteBulkOp struct {
	Delete deleteBulkOpData `json:"delete"`
}

type deleteBulkOpData struct {
	Id            string `json:"_id"`
	Index         string `json:"_index"`
	IfSeqNo       uint64 `json:"if_seq_no"`
	IfPrimaryTerm uint64 `json:"if_primary_term"`
}

type insertArchivedTaskOpPair struct {
	op  insertArchivedTaskOp
	doc PersistedTaskData
}

type insertArchivedTaskOp struct {
	Index insertArchivedTaskOpData `json:"index"`
}

type insertArchivedTaskOpData struct {
	Index string `json:"_index"`
}

type tasksWithScrollId struct {
	ScrollId string
	Tasks    []task.Task
}

type esSearchScrollingResponse struct {
	Hits struct {
		Hits []esHitPersistedTask `json:"hits"`
	} `json:"hits"`
	ScrollId string `json:"_scroll_id"`
}

func buildTimedOutSearchBody(nowUtc time.Time, pageSize uint) jsonObjMap {
	return jsonObjMap{
		"size":                pageSize,
		"seq_no_primary_term": true,
		"sort": []jsonObjMap{
			{
				"run_at": jsonObjMap{
					"order": "asc",
				},
			},
		},
		"query": jsonObjMap{
			"bool": jsonObjMap{
				"filter": jsonObjMap{
					"bool": jsonObjMap{
						"must": []jsonObjMap{
							{
								"range": jsonObjMap{
									"last_claimed.times_out_at": jsonObjMap{
										"lte": nowUtc.Format(time.RFC3339Nano),
									},
								},
							}, {
								"term": jsonObjMap{
									"state": task.CLAIMED.String(), // just CLAIMED
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildArchivableSearchBody(archiveFinishedBefore time.Time, pageSize uint) jsonObjMap {
	return jsonObjMap{
		"size":                pageSize,
		"seq_no_primary_term": true,
		"sort": []jsonObjMap{
			{
				"last_claimed.result.at": jsonObjMap{
					"order": "asc",
				},
			},
			{
				"run_at": jsonObjMap{
					"order": "asc",
				},
			},
			{
				"metadata.created_at": jsonObjMap{
					"order": "asc",
				},
			},
		},
		"query": jsonObjMap{
			"bool": jsonObjMap{
				"filter": jsonObjMap{
					"bool": jsonObjMap{
						"must": []jsonObjMap{
							{
								"range": jsonObjMap{
									"last_claimed.result.at": jsonObjMap{
										"lte": archiveFinishedBefore.Format(time.RFC3339Nano),
									},
								},
							}, {
								"bool": jsonObjMap{
									"should": []jsonObjMap{
										{
											"term": jsonObjMap{
												"state": task.DONE.String(),
											},
										},
										{
											"term": jsonObjMap{
												"state": task.DEAD.String(),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildOutstandingTasksCountQuery(recurringTaskId task.RecurringTaskId) jsonObjMap {
	return jsonObjMap{
		"query": jsonObjMap{
			"bool": jsonObjMap{
				"filter": jsonObjMap{
					"bool": jsonObjMap{
						"must": []jsonObjMap{
							{
								"term": jsonObjMap{
									"recurring_task_id.keyword": string(recurringTaskId),
								},
							},
							{
								"bool": jsonObjMap{
									"should": []jsonObjMap{
										{
											"term": jsonObjMap{
												"state": task.QUEUED.String(), // just queued
											},
										},
										{
											"term": jsonObjMap{
												"state": task.FAILED.String(), // we can retry
											},
										},
										{
											"term": jsonObjMap{
												"state": task.CLAIMED.String(), // is claimed
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
