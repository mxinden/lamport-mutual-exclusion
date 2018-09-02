package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

func init() {
	rand.Seed(42)
}

func main() {
	count := 3
	processes := []*process{}
	done := make(chan struct{})
	defer close(done)

	for i := 0; i < count; i++ {
		processes = append(
			processes,
			newProcess(i, done),
		)
	}

	for i, p := range processes {
		if i == 0 {
			p.registerPeers(processes[1:])
			continue
		}

		if i == len(processes)-1 {
			p.registerPeers(processes[0:i])
			continue
		}

		p.registerPeers(
			append(
				append([]*process{}, processes[0:i]...),
				processes[i+1:]...,
			),
		)
	}

	for _, p := range processes {
		go p.run()
	}

	time.Sleep(10 * time.Second)
}

type messageQueue []message

type byTime messageQueue

func (s byTime) Len() int { return len(s) }

func (s byTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s byTime) Less(i, j int) bool {
	if s[i].getTimestamp() == s[j].getTimestamp() {
		return s[i].getSource() < s[j].getSource()
	}

	return s[i].getTimestamp() < s[j].getTimestamp()
}

type process struct {
	name         int
	messageQueue messageQueue
	time         int
	peers        []*process
	receiveC     chan message
	done         <-chan struct{}
}

func newProcess(n int, done <-chan struct{}) *process {
	return &process{
		name:         n,
		messageQueue: messageQueue{},
		time:         0,
		peers:        []*process{},
		// TODO: Remove static 3
		receiveC: make(chan message, 3),
		done:     done,
	}
}

func (p *process) registerPeers(peers []*process) {
	p.peers = peers
}

func (p *process) run() {
	fmt.Printf(">> process %v: run\n", p.name)
	for {
		select {
		case <-p.done:
			fmt.Printf(">> process %v: done\n", p.name)
			return
		case m := <-p.receiveC:
			p.receive(m)
		case <-time.After(time.Duration(rand.Intn(5)) * time.Second):
			p.request()
		}
	}
}

func (p *process) hasResource() bool {
	sort.Sort(byTime(p.messageQueue))

	requestFound := false
	var requestFoundAt int

	for i, m := range p.messageQueue {
		req, ok := m.(*request)
		if !ok {
			continue
		}

		if req.source != p.name {
			// someone else made the request before us
			return false
		}

		requestFound = true
		requestFoundAt = i
		break
	}

	if !requestFound {
		return false
	}

	for _, peer := range p.peers {
		peerFound := false

		for i := requestFoundAt + 1; i < len(p.messageQueue); i++ {
			m := p.messageQueue[i]
			ack, ok := m.(*acknowledgement)
			if !ok {
				continue
			}

			if ack.getSource() != peer.name {
				continue
			}

			peerFound = true
		}

		if !peerFound {
			return false
		}
	}

	return true
}

func (p *process) release() {
	fmt.Printf(">> process %v: release\n", p.name)

	p.time = p.time + 1

	p.cleanMessageQueue()

	r := release{p.name, p.time}

	p.broadcast(&r)
}

func (p *process) cleanMessageQueue() {
	fmt.Printf(">> process %v: cleanRequestQueue\n", p.name)

	firstRequestAt := 0

	for i, m := range p.messageQueue {
		_, ok := m.(*request)

		if ok {
			firstRequestAt = i
			break
		}
	}

	p.messageQueue = p.messageQueue[firstRequestAt+1:]
}

func (p *process) request() {
	fmt.Printf(">> process %v: request\n", p.name)

	p.time = p.time + 1

	m := request{
		source:    p.name,
		timestamp: p.time,
	}

	p.messageQueue = append(p.messageQueue, &m)

	p.broadcast(&m)
}

func max(a, b int) int {
	if a < b {
		return b
	}

	return a
}

func (p *process) receive(m message) {
	p.time = max(p.time, m.getTimestamp()) + 1

	switch msg := m.(type) {
	case *request:
		fmt.Printf(">> process %v: received request message %v\n", p.name, m)
		p.messageQueue = append(p.messageQueue, m)
		p.acknowledge(msg)
	case *acknowledgement:
		fmt.Printf(">> process %v: received acknowledgement message %v\n", p.name, m)
		p.messageQueue = append(p.messageQueue, m)
		if p.hasResource() {
			fmt.Printf("## process %v: has resource\n", p.name)
			p.release()
		}
	case *release:
		fmt.Printf(">> process %v: received release message %v\n", p.name, m)
		p.cleanMessageQueue()
	}
}

func (p *process) acknowledge(m *request) {
	fmt.Printf(">> process %v: acknowledge request message %v\n", p.name, m)

	p.time = p.time + 1

	ack := acknowledgement{p.name, p.time}

	for _, peer := range p.peers {
		if peer.name == m.source {
			peer.receiveC <- &ack
			return
		}
	}
}

func (p *process) broadcast(m message) {
	for _, p := range p.peers {
		p.receiveC <- m
	}
}

type message interface {
	getSource() int
	getTimestamp() int
}

type request struct {
	source    int
	timestamp int
}

func (r *request) getSource() int    { return r.source }
func (r *request) getTimestamp() int { return r.timestamp }

type acknowledgement struct {
	source    int
	timestamp int
}

func (r *acknowledgement) getSource() int    { return r.source }
func (r *acknowledgement) getTimestamp() int { return r.timestamp }

type release struct {
	source    int
	timestamp int
}

func (r *release) getSource() int    { return r.source }
func (r *release) getTimestamp() int { return r.timestamp }
