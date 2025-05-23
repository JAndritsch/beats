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

package udp

import (
	"fmt"
	"net"
	"strconv"

	"github.com/elastic/beats/v7/metricbeat/helper/server"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type UdpServer struct {
	udpaddr           *net.UDPAddr
	listener          *net.UDPConn
	receiveBufferSize int
	done              chan struct{}
	eventQueue        chan server.Event
	logger            *logp.Logger
}

type UdpEvent struct {
	event mapstr.M
	meta  server.Meta
}

func (u *UdpEvent) GetEvent() mapstr.M {
	return u.event
}

func (u *UdpEvent) GetMeta() server.Meta {
	return u.meta
}

func NewUdpServer(base mb.BaseMetricSet) (server.Server, error) {
	config := defaultUdpConfig()
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(config.Host, strconv.Itoa(config.Port)))

	if err != nil {
		return nil, err
	}

	return &UdpServer{
		udpaddr:           addr,
		receiveBufferSize: config.ReceiveBufferSize,
		done:              make(chan struct{}),
		eventQueue:        make(chan server.Event),
		logger:            base.Logger(),
	}, nil
}

func (g *UdpServer) GetHost() string {
	return g.udpaddr.String()
}

func (g *UdpServer) Start() error {
	listener, err := net.ListenUDP("udp", g.udpaddr)
	if err != nil {
		return fmt.Errorf("failed to start UDP server: %w", err)
	}

	g.logger.Infof("Started listening for UDP on: %s", g.udpaddr.String())
	g.listener = listener

	go g.watchMetrics()
	return nil
}

func (g *UdpServer) watchMetrics() {
	buffer := make([]byte, g.receiveBufferSize)
	for {
		select {
		case <-g.done:
			return
		default:
		}

		length, addr, err := g.listener.ReadFromUDP(buffer)
		if err != nil {
			g.logger.Errorf("Error reading from buffer: %v", err.Error())
			continue
		}

		bufCopy := make([]byte, length)
		copy(bufCopy, buffer)

		g.eventQueue <- &UdpEvent{
			event: mapstr.M{
				server.EventDataKey: bufCopy,
			},
			meta: server.Meta{
				"client_ip": addr.IP.String(),
			},
		}
	}
}

func (g *UdpServer) GetEvents() chan server.Event {
	return g.eventQueue
}

func (g *UdpServer) Stop() {
	close(g.done)
	g.listener.Close()
	close(g.eventQueue)
}
