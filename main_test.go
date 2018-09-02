package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestHasResource(t *testing.T) {
	p := process{
		name: 2,
		peers: []*process{
			&process{name: 0},
			&process{name: 1},
		},
		messageQueue: []message{
			&request{source: 2, timestamp: 51},
			&request{source: 1, timestamp: 52},
			&acknowledgement{source: 0, timestamp: 53},
			&acknowledgement{source: 1, timestamp: 54},
		},
	}

	if !p.hasResource() {
		t.Fatal("expected process to have resource")
	}
}

func (p *process) printQueue() {
	queue := []string{}
	for _, m := range p.messageQueue {
		item := ""
		switch m.(type) {
		case *request:
			item = item + "req"
		case *acknowledgement:
			item = item + "ack"
		case *release:
			item = item + "rel"
		}
		item = item + "(" + strconv.Itoa(m.getSource()) + "," + strconv.Itoa(m.getTimestamp()) + ")"

		queue = append(queue, item)
	}

	fmt.Printf(">> process %v: queue: %v\n", p.name, strings.Join(queue, "; "))
}

func TestCleanRequestQueue(t *testing.T) {
	p := process{
		name: 1,
		peers: []*process{
			&process{name: 0},
			&process{name: 2},
		},
		messageQueue: []message{
			&request{source: 2, timestamp: 35},
			&request{source: 1, timestamp: 38},
			&request{source: 2, timestamp: 50},
		},
	}

	p.cleanMessageQueue()

	p.printQueue()

	expected := 2
	if len(p.messageQueue) != expected {
		t.Fatalf("expected process 1 to have %v messages in its queue but got %v", expected, len(p.messageQueue))
	}
}
