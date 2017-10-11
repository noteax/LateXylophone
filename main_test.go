package main

import (
	"testing"
	"time"
	"sync"
	"fmt"
)

func TestRequestProcessing(t *testing.T) {

	var lb LoadBalancer = &MyLoadBalancer{
		disableInterval: 1 * time.Minute,
		responseTimeout: 4 * time.Second,
		mutex:           &sync.Mutex{},
		instances:       []*Instance{},
	}

	// spawn instances
	manager := &TimeServiceManager{}
	for i := 0; i < 50; i++ {
		ts := manager.Spawn()
		lb.RegisterInstance(ts.ReqChan)
		go ts.Run()
	}

	// kill some of them
	for i := 0; i < 25; i++ {
		manager.Kill()
	}

	var wg sync.WaitGroup
	wg.Add(80)

	// send requests
	responses := make(chan Response, 80)
	timeouts := make(chan Response, 80)
	for i := 0; i < 80; i++ {
		go func() {
			defer wg.Done()
			select {
			case rsp := <-lb.Request(nil):
				fmt.Printf("Time: %s\n", rsp)
				responses <- rsp
			case <-time.After(16 * time.Second):
				timeouts <- "timeout"
			}
		}()
	}

	// wait for all requests completion
	wg.Wait()

	// check that there is no timeouts at this configuration
	if 80 != len(responses) {
		t.Error("Failed to get response to all 100 requests. Successful only", len(responses))
	}
}
