package leader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/internal/domain/leader"
	"github.com/lloydmeta/tasques/internal/domain/tracing"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
)

var (
	IndexName = common.IndexName(".tasques_leader_locks")
)

type processId string

type state uint32

func (s state) String() string {
	return statesToString[s]
}

const (
	CHECKER state = iota
	PRETENDER
	USURPER
	LEADER
	STOPPED
)

var statesToString = map[state]string{
	CHECKER:   "CHECKER",
	PRETENDER: "PRETENDER",
	USURPER:   "USURPER",
	LEADER:    "LEADER",
	STOPPED:   "STOPPED",
}

/*
 * EsLock implements a very basic state machine for checking if the current `EsLock` process
 * is the leader based on what is stored inside the document in the leader index, as identified
 * by ID.
 *
 * It does this by using a simple polling mechanism along with a looping FSM that makes use of
 * The concurrent locking mechanisms in ES to figure out it should attempt to become the leader.
 *
 * Limitations:
 *  - Not real time (polling..)
 *  - Depends on not having too much drift in machine clock between all servers (time diff).
 */
type EsLock struct {
	leaderLockDockId common.DocumentID

	processId processId
	client    *elasticsearch.Client
	getUTC    func() time.Time // for mocking
	state     state            // set and get via atomic ops since we'll access this from different threads

	loopInterval             time.Duration
	leaderReportLagTolerance time.Duration

	stashedDoc *esLeaderInfo // could be empty

	tracer    tracing.Tracer
	stateLock sync.Mutex // used only when we modify the state
}

// Ignore: this is for tests
func (e *EsLock) SetUTCGetter(getter func() time.Time) {
	e.getUTC = getter
}

func buildProcessId(id common.DocumentID) processId {
	uniqueId := strings.ReplaceAll(uuid.New().String(), "-", "")
	return processId(fmt.Sprintf("%s-%s", string(id), uniqueId))
}

// NewLeaderLock returns a new leader.Lock
//
// Generates a random process id for the returned instance.
func NewLeaderLock(leaderLockDocId common.DocumentID, client *elasticsearch.Client, loopInterval time.Duration, leaderReportLagTolerance time.Duration, tracer tracing.Tracer) leader.Lock {
	return &EsLock{
		leaderLockDockId: leaderLockDocId,
		processId:        buildProcessId(leaderLockDocId),
		client:           client,
		getUTC: func() time.Time {
			return time.Now().UTC()
		},
		state:                    CHECKER,
		loopInterval:             loopInterval,
		leaderReportLagTolerance: leaderReportLagTolerance,
		stashedDoc:               nil,
		stateLock:                sync.Mutex{},
		tracer:                   tracer,
	}
}

func (e *EsLock) IsLeader() bool {
	return e.getState() == LEADER
}

func (e *EsLock) getState() state {
	return state(atomic.LoadUint32((*uint32)(&e.state)))
}
func (e *EsLock) setState(newState state) {
	if oldState := state(atomic.SwapUint32((*uint32)(&e.state), uint32(newState))); oldState != newState {
		log.Info().
			Str("old_state", oldState.String()).
			Str("new_state", newState.String()).
			Str("process_id", string(e.processId)).
			Msg("Setting State")
	}
}

func (e *EsLock) getLeaderDoc() (*esLeaderInfo, error) {
	tx := e.tracer.BackgroundTx("leader-lock-getLeaderDoc")
	defer tx.End()
	ctx := tx.Context()
	getReq := esapi.GetRequest{

		Index:      string(IndexName),
		DocumentID: string(e.leaderLockDockId),
	}
	rawResp, err := getReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	switch rawResp.StatusCode {
	case 200:
		var info esLeaderInfo
		if err := json.NewDecoder(rawResp.Body).Decode(&info); err != nil {
			return nil, common.JsonSerdesErr{Underlying: []error{err}}
		}
		return &info, nil
	case 404:
		return nil, NotFound{}
	default:
		return nil, common.UnexpectedEsStatusError(rawResp)
	}
}

func (e *EsLock) submitNameForLeader() (*esLeaderInfo, error) {
	tx := e.tracer.BackgroundTx("leader-lock-submitNameForLeader")
	defer tx.End()
	ctx := tx.Context()

	now := e.getUTC()
	data := leaderData{
		LeaderId: e.processId,
		At:       now,
	}
	dataAsBytes, err := json.Marshal(data)
	if err != nil {
		return nil, common.JsonSerdesErr{Underlying: []error{err}}
	}

	createReq := esapi.CreateRequest{
		Index:      string(IndexName),
		DocumentID: string(e.leaderLockDockId),
		Body:       bytes.NewReader(dataAsBytes),
	}

	rawResp, err := createReq.Do(ctx, e.client)
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
		info := esLeaderInfo{
			ID:          response.ID,
			SeqNum:      seqNum(response.SeqNum),
			PrimaryTerm: primaryTerm(response.PrimaryTerm),
			Source: leaderData{
				LeaderId: e.processId,
				At:       now,
			},
		}
		return &info, nil
	case statusCode == 409:
		return nil, Conflict{}
	default:
		return nil, common.UnexpectedEsStatusError(rawResp)
	}

}

// Tries to update the doc in ES so that the current lock is the leader
func (e *EsLock) jostleForLeader(primaryT primaryTerm, seqNo seqNum) (*esLeaderInfo, error) {
	tx := e.tracer.BackgroundTx("leader-lock-getLeaderDoc")
	defer tx.End()
	ctx := tx.Context()

	now := e.getUTC()
	data := leaderData{
		LeaderId: e.processId,
		At:       now,
	}
	dataAsBytes, err := json.Marshal(data)
	if err != nil {
		return nil, common.JsonSerdesErr{Underlying: []error{err}}
	}
	// Purposely using the Index API (rather than the update API) so as to
	// not get bit by old stale data due to partial updates. We send optimistic
	// locking data to ensure we are _updating_
	updateReq := esapi.IndexRequest{
		Index:         string(IndexName),
		DocumentID:    string(e.leaderLockDockId),
		Body:          bytes.NewReader(dataAsBytes),
		IfPrimaryTerm: esapi.IntPtr(int(primaryT)),
		IfSeqNo:       esapi.IntPtr(int(seqNo)),
	}

	rawResp, err := updateReq.Do(ctx, e.client)
	if err != nil {
		return nil, common.ElasticsearchErr{Underlying: err}
	}
	defer rawResp.Body.Close()
	statusCode := rawResp.StatusCode
	switch {
	case 200 <= statusCode && statusCode <= 299:
		// Updated, grab new metadata
		var resp common.EsUpdateResponse
		if err := json.NewDecoder(rawResp.Body).Decode(&resp); err != nil {
			return nil, common.JsonSerdesErr{Underlying: []error{err}}
		}
		info := esLeaderInfo{
			ID:          resp.ID,
			SeqNum:      seqNum(resp.SeqNum),
			PrimaryTerm: primaryTerm(resp.PrimaryTerm),
			Source: leaderData{
				LeaderId: e.processId,
				At:       now,
			},
		}
		return &info, nil
	case statusCode == 404:
		return nil, NotFound{}
	case statusCode == 409:
		return nil, Conflict{}
	default:
		return nil, common.UnexpectedEsStatusError(rawResp)
	}
}

// This is the basic loop for our state machine. Written as one big unbroken, function because
// it feels easier to see all the logic up front when it's like this. Oh, and GOTO because we can.
func (e *EsLock) loop() {
top:
	e.stateLock.Lock()
	loopStartTime := e.getUTC()
	ignoreMinLoopInterval := false
	switch e.getState() {
	case LEADER:
		if e.stashedDoc != nil {
			if r, err := e.jostleForLeader(e.stashedDoc.PrimaryTerm, e.stashedDoc.SeqNum); err != nil {
				switch v := err.(type) {
				case NotFound:
					ignoreMinLoopInterval = true
					e.stashedDoc = nil
					e.setState(PRETENDER)
				case Conflict:
					e.stashedDoc = nil
					e.setState(CHECKER)
				default:
					log.Error().Err(v).Msg("Unexpected error, ignoring for now")
					e.stashedDoc = nil
					e.setState(CHECKER)
				}
			} else {
				e.stashedDoc = r
				e.setState(LEADER)
			}
		} else { // impossible, but defensively doing this.
			e.setState(CHECKER)
		}
	case PRETENDER:
		if r, err := e.submitNameForLeader(); err != nil {
			switch v := err.(type) {
			case Conflict:
				e.stashedDoc = nil
				e.setState(CHECKER)
			default:
				log.Error().Err(v).Msg("Unexpected error, ignoring for now")
				e.stashedDoc = nil
				e.setState(CHECKER)
			}
		} else {
			e.stashedDoc = r
			e.setState(LEADER)
		}
	case CHECKER:
		if r, err := e.getLeaderDoc(); err != nil {
			switch v := err.(type) {
			case NotFound:
				ignoreMinLoopInterval = true
				e.stashedDoc = nil
				e.setState(PRETENDER)
			default:
				log.Error().Err(v).Msg("Unexpected error, ignoring for now")
				e.stashedDoc = nil
				e.setState(CHECKER)
			}
		} else {
			now := e.getUTC()
			if now.Sub(r.Source.At) > e.leaderReportLagTolerance {
				log.Debug().Msgf("now [%v] doc at [%v]", now, r.Source.At)
				log.Debug().Msgf("too much lag; got [%v] > [%v]", now.Sub(r.Source.At), e.leaderReportLagTolerance)
				ignoreMinLoopInterval = true
				e.stashedDoc = r
				e.setState(USURPER)
			} else {
				// still reliable
				if r.Source.LeaderId == e.processId {
					ignoreMinLoopInterval = true // re-enforce
					e.stashedDoc = r
					e.setState(LEADER)
				} else {
					e.stashedDoc = nil
					e.setState(CHECKER)
				}
			}
		}
	case USURPER:
		if e.stashedDoc == nil {
			log.Error().Msg("Could not find stashed leader doc when usurping leader.")
			e.setState(CHECKER)
		} else {
			if r, err := e.jostleForLeader(e.stashedDoc.PrimaryTerm, e.stashedDoc.SeqNum); err != nil {
				switch v := err.(type) {
				case Conflict:
					e.stashedDoc = nil
					e.setState(CHECKER)
				case NotFound:
					ignoreMinLoopInterval = true
					e.stashedDoc = nil
					e.setState(PRETENDER)
				default:
					log.Error().Err(v).Msg("Unexpected error, ignoring for now")
					e.stashedDoc = nil
					e.setState(CHECKER)
				}
			} else {
				ignoreMinLoopInterval = true // re-enforce
				e.stashedDoc = r
				e.setState(LEADER)
			}
		}
	case STOPPED:
		e.stateLock.Unlock()
		return // exit
	default:
		e.stashedDoc = nil
		e.setState(CHECKER)
	}
	e.stateLock.Unlock()
	if !ignoreMinLoopInterval {
		loopEndTime := e.getUTC()
		waitTime := e.loopInterval - loopEndTime.Sub(loopStartTime)
		if waitTime > 0 {
			time.Sleep(waitTime)
		}
	}
	goto top
}

func (e *EsLock) Start() {
	e.stateLock.Lock()
	defer e.stateLock.Unlock()
	e.setState(CHECKER)
	go e.loop()
}
func (e *EsLock) Stop() {
	e.stateLock.Lock()
	defer e.stateLock.Unlock()
	e.setState(STOPPED)
}

type NotFound struct{}

func (n NotFound) Error() string {
	return "Not the leader"
}

type Conflict struct{}

func (n Conflict) Error() string {
	return "Leader doc exists"
}

type leaderData struct {
	LeaderId processId `json:"leader_id"`
	At       time.Time `json:"at"`
}

type primaryTerm uint64
type seqNum uint64

type esLeaderInfo struct {
	ID          string      `json:"_id"`
	SeqNum      seqNum      `json:"_seq_no"`
	PrimaryTerm primaryTerm `json:"_primary_term"`
	Source      leaderData  `json:"_source"`
}
