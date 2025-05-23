// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package diskqueue

import (
	"flag"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

var seed int64

type testQueue struct {
	*diskQueue
}

func init() {
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "test random seed")
}

func TestProduceConsumer(t *testing.T) {
	maxEvents := 1024
	minEvents := 32

	r := rand.New(rand.NewPCG(uint64(seed), 0)) //nolint:gosec //Safe to ignore in tests
	events := r.IntN(maxEvents-minEvents) + minEvents
	batchSize := r.IntN(events-8) + 4
	bufferSize := r.IntN(batchSize*2) + 4

	// events := 4
	// batchSize := 1
	// bufferSize := 2

	t.Log("seed: ", seed)
	t.Log("events: ", events)
	t.Log("batchSize: ", batchSize)
	t.Log("bufferSize: ", bufferSize)

	testWith := func(factory queuetest.QueueFactory) func(t *testing.T) {
		return func(t *testing.T) {
			t.Run("single", func(t *testing.T) {
				t.Parallel()
				queuetest.TestSingleProducerConsumer(t, events, batchSize, factory)
			})
			t.Run("multi", func(t *testing.T) {
				t.Parallel()
				queuetest.TestMultiProducerConsumer(t, events, batchSize, factory)
			})
		}
	}

	t.Run("direct", testWith(makeTestQueue()))
}

func makeTestQueue() queuetest.QueueFactory {
	return func(t *testing.T) queue.Queue {
		dir := t.TempDir()
		settings := DefaultSettings()
		settings.Path = dir
		logger := logptest.NewTestingLogger(t, "")
		queue, _ := NewQueue(logger, nil, settings, nil)
		return testQueue{
			diskQueue: queue,
		}
	}
}

func (t testQueue) Close() error {
	err := t.diskQueue.Close()
	return err
}
