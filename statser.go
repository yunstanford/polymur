package main

import (
	"log"
	"time"
)

type Statser struct {
	value chan int64
}

func NewStatser() *Statser {
	s := &Statser{make(chan int64, 1)}
	s.init()
	return s
}

func (s *Statser) init() {
	s.value <- 0
}

func (s *Statser) IncrRecv(v int64) {
	i := <-s.value
	s.value <- i + v
}

func (s *Statser) FetchRecv() int64 {
	i := <-s.value
	s.value <- i
	return i
}

// Outputs periodic info summary.
func statsTracker(s *Statser) {
	tick := time.Tick(5 * time.Second)
	var currCnt, lastCnt int64

	for {
		<-tick

		// Inbound rates.
		lastCnt = currCnt
		currCnt = s.FetchRecv()
		deltaCnt := currCnt - lastCnt
		if deltaCnt > 0 {
			log.Printf("Last 5s: Received %d data points | Avg: %.2f/sec. | Inbound queue length: %d\n",
				deltaCnt,
				float64(deltaCnt)/5,
				len(messageIncomingQueue))
		}

		// Outbound queues.
		pool.Lock()
		for dest, outboundQueue := range pool.Conns {
			currLen := len(outboundQueue)
			switch {
			case currLen == options.queuecap:
				log.Printf("Destination %s queue is at capacity (%d) - further messages will be dropped", dest, currLen)
			case currLen > 0:
				log.Printf("Destination %s queue length: %d\n", dest, currLen)
			}
		}
		pool.Unlock()

		// Misc. internal queues.
		if l := len(failedMessages); l > 0 {
			log.Printf("Failed message queue length: %d\n", l)
		}

	}
}
