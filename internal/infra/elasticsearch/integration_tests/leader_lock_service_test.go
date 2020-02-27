// +build integration

package integration_tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	leader2 "github.com/lloydmeta/tasques/internal/domain/leader"
	"github.com/lloydmeta/tasques/internal/infra/apm/tracing"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/common"
	"github.com/lloydmeta/tasques/internal/infra/elasticsearch/leader"
)

func defaultCreateLock(lockName string) leader2.Lock {
	return leader.NewLeaderLock(common.DocumentID(lockName), esClient, 250*time.Millisecond, 500*time.Millisecond, tracing.NoopTracer{})
}

func Test_EsLeaderLock_Create(t *testing.T) {
	lock := defaultCreateLock("leader_lock1")
	assert.False(t, lock.IsLeader())
}

func Test_EsLeaderLock_Starting_and_Stopping(t *testing.T) {
	lock := defaultCreateLock("leader_lock2")
	assert.False(t, lock.IsLeader())

	lock.Start()

	assert.Eventually(t, func() bool {
		return lock.IsLeader()
	}, 3*time.Second, 300*time.Millisecond)

	lock.Stop()

	assert.False(t, lock.IsLeader())
}

func Test_EsLeaderLock_Starting_and_Stopping_Multiple(t *testing.T) {
	lock1 := defaultCreateLock("leader_lock3")
	lock2 := defaultCreateLock("leader_lock3")
	lock3 := defaultCreateLock("leader_lock3")

	lock1.Start()

	assert.Eventually(t, func() bool {
		return lock1.IsLeader()
	}, 3*time.Second, 300*time.Millisecond)

	// Test for 1000 iterations
	for i := 0; i < 1000; i++ {
		assert.True(t, lock1.IsLeader())
	}

	// start the other ones
	lock2.Start()
	lock3.Start()

	// Test for 1000 iterations
	for i := 0; i < 1000; i++ {
		assert.True(t, lock1.IsLeader())
	}

	assert.Eventually(t, func() bool {
		return lock1.IsLeader() != lock2.IsLeader() != lock3.IsLeader()
	}, 5*time.Second, 300*time.Millisecond, "Wait for XOR isLeader")

	// Test for 1000 iterations
	for i := 0; i < 1000; i++ {
		assert.True(t, lock1.IsLeader() != lock2.IsLeader() != lock3.IsLeader(), "XOR isLeader")
	}

	// kill all but one
	lock1.Stop()
	assert.False(t, lock1.IsLeader())
	lock2.Stop()
	assert.False(t, lock2.IsLeader())
	assert.Eventually(t, func() bool {
		return lock3.IsLeader()
	}, 5*time.Second, 300*time.Millisecond, "The survivor should become the leader")
	lock3.Stop()
}
