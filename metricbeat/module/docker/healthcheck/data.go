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

package healthcheck

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/container"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func eventsMapping(r mb.ReporterV2, containers []container.Summary, m *MetricSet) {
	for _, container := range containers {
		eventMapping(r, &container, m)
	}
}

func eventMapping(r mb.ReporterV2, cont *container.Summary, m *MetricSet) {
	if !hasHealthCheck(cont.Status) {
		return
	}

	container, err := m.dockerClient.ContainerInspect(context.TODO(), cont.ID)
	if err != nil {
		return
	}

	// Check if the container has any health check
	if container.State.Health == nil {
		return
	}

	lastEvent := len(container.State.Health.Log) - 1

	// Checks if a healthcheck already happened
	if lastEvent < 0 {
		return
	}

	fields := mapstr.M{
		"status":        container.State.Health.Status,
		"failingstreak": container.State.Health.FailingStreak,
		"event": mapstr.M{
			"start_date": common.Time(container.State.Health.Log[lastEvent].Start),
			"end_date":   common.Time(container.State.Health.Log[lastEvent].End),
			"exit_code":  container.State.Health.Log[lastEvent].ExitCode,
			"output":     container.State.Health.Log[lastEvent].Output,
		},
	}

	r.Event(mb.Event{
		RootFields:      docker.NewContainer(cont, m.dedot).ToMapStr(),
		MetricSetFields: fields,
	})
}

// hasHealthCheck detects if healthcheck is available for container
func hasHealthCheck(status string) bool {
	return strings.Contains(status, "(") && strings.Contains(status, ")")
}
