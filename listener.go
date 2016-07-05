// The MIT License (MIT)
//
// Copyright (c) 2016 Jamie Alquiza
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
package main

import (
	"bufio"
	"log"
	"net"
	"time"
)

func init() {
	config.batchSize = 30
	config.flushTimeout = 5
}

// Listens for messages.
func listener(s *Statser) {
	log.Printf("Metrics listener started: %s:%s\n",
		options.addr,
		options.port)
	server, err := net.Listen("tcp", options.addr+":"+options.port)
	if err != nil {
		log.Fatalf("Listener error: %s\n", err)
	}
	defer server.Close()

	// Connection handler loop.
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Printf("Connection handler error: %s\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		go connectionHandler(conn, s)
	}
}

func connectionHandler(c net.Conn, s *Statser) {
	flushTimeout := time.NewTicker(time.Duration(config.flushTimeout) * time.Second)
	defer flushTimeout.Stop()

	messages := []*string{}

	inbound := bufio.NewScanner(c)
	defer c.Close()

	for inbound.Scan() {

		// We hit the flush timeout, load the current batch if present.
		select {
		case <-flushTimeout.C:
			if len(messages) > 0 {
				messageIncomingQueue <- messages
				messages = []*string{}
			}
			messages = []*string{}
		default:
			break
		}

		m := inbound.Text()
		s.UpdateCount(1)

		// Drop message and respond if the incoming queue is at capacity.
		if len(messageIncomingQueue) >= options.queuecap {
			log.Printf("Queue capacity %d reached\n", options.queuecap)
			// Impose flow control. This needs to be significantly smarter.
			time.Sleep(1 * time.Second)
		}

		// If this puts us at the batchSize threshold, enqueue
		// into the messageIncomingQueue.
		if len(messages)+1 >= config.batchSize {
			messages = append(messages, &m)
			messageIncomingQueue <- messages
			messages = []*string{}
		} else {
			// Otherwise, just append message to current batch.
			messages = append(messages, &m)
		}

	}

	messageIncomingQueue <- messages

}
