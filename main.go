package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
	"sync"
)

// Request is used for making requests to services behind a load balancer.
type Request struct {
	Payload interface{}
	RspChan chan Response
}

// Response is the value returned by services behind a load balancer.
type Response interface{}

// LoadBalancer is used for balancing load between multiple instances of a service.
type LoadBalancer interface {
	Request(payload interface{}) chan Response
	RegisterInstance(chan Request)
}

// Implementation part

type MyLoadBalancer struct {
	disableInterval time.Duration
	responseTimeout time.Duration

	mutex     *sync.Mutex
	instances []*Instance
	index     int
}

type Instance struct {
	disabledUntil time.Time
	ch            chan Request
}

// Request is currently a dummy implementation. Please implement it!
func (lb *MyLoadBalancer) Request(payload interface{}) chan Response {
	ch := make(chan Response, 1)

	lb.mutex.Lock()
	initialInstance, err := lb.nextAliveInstance()
	lb.mutex.Unlock()

	if err != nil {
		ch <- "Request can not be processed"
		return ch
	}

	go func() {
		instance := initialInstance
		triedInstances := make(map[*Instance]struct{})
	L:
		for {
			triedInstances[instance] = struct{}{}

			intermediate := make(chan Response, 1)
			instance.ch <- Request{Payload: payload, RspChan: intermediate}

			select {
			case res := <-intermediate:
				ch <- res
				break L
			case <-time.After(lb.responseTimeout):
				// mark as not alive for a while and try next alive instance
				lb.mutex.Lock()
				instance.disabledUntil = time.Now().Add(lb.disableInterval)
				instance, err = lb.nextAliveInstance()
				_, tried := triedInstances[instance]
				if err != nil || tried {
					ch <- "Request can not be processed"
				}
				lb.mutex.Unlock()
			}
		}
	}()
	return ch
}

func (lb *MyLoadBalancer) nextAliveInstance() (*Instance, error) {
	currentLbIndex := lb.index
	for i := currentLbIndex; i < currentLbIndex+len(lb.instances); i++ {
		slicedIndex := i % len(lb.instances)
		instance := lb.instances[slicedIndex]
		if time.Now().Sub(instance.disabledUntil) >= 0 {
			lb.index++
			return instance, nil
		}
	}
	return nil, fmt.Errorf("No alive instances")
}

// RegisterInstance is currently a dummy implementation. Please implement it!
func (lb *MyLoadBalancer) RegisterInstance(ch chan Request) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.instances = append(lb.instances, &Instance{
		time.Time{},
		ch,
	})
}

/******************************************************************************
 *  STANDARD TIME SERVICE IMPLEMENTATION -- MODIFY IF YOU LIKE                *
 ******************************************************************************/

// TimeService is a single instance of a time service.
type TimeService struct {
	Dead            chan struct{}
	ReqChan         chan Request
	AvgResponseTime float64
}

// Run will make the TimeService start listening to the two channels Dead and ReqChan.
func (ts *TimeService) Run() {
	for {
		select {
		case <-ts.Dead:
			return
		case req := <-ts.ReqChan:
			processingTime := time.Duration(ts.AvgResponseTime+1.0-rand.Float64()) * time.Second
			time.Sleep(processingTime)
			req.RspChan <- time.Now()
		}
	}
}

/******************************************************************************
 *  CLI -- YOU SHOULD NOT NEED TO MODIFY ANYTHING BELOW                       *
 ******************************************************************************/

// main runs an interactive console for spawning, killing and asking for the
// time.
func main() {
	rand.Seed(int64(time.Now().Nanosecond()))

	bio := bufio.NewReader(os.Stdin)

	var lb LoadBalancer = &MyLoadBalancer{
		disableInterval: 1 * time.Minute,
		responseTimeout: 4 * time.Second,
		mutex:           &sync.Mutex{},
		instances:       []*Instance{},
	}

	manager := &TimeServiceManager{}

	for {
		fmt.Printf("> ")
		cmd, err := bio.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading command: ", err)
			continue
		}
		switch strings.TrimSpace(cmd) {
		case "kill":
			manager.Kill()
		case "spawn":
			ts := manager.Spawn()
			lb.RegisterInstance(ts.ReqChan)
			go ts.Run()
		case "time":
			select {
			case rsp := <-lb.Request(nil):
				fmt.Println(rsp)
			case <-time.After(7 * time.Second):
				fmt.Println("Timeout")
			}
		default:
			fmt.Printf("Unknown command: %s Available commands: time, spawn, kill\n", cmd)
		}
	}
}

// TimeServiceManager is responsible for spawning and killing.
type TimeServiceManager struct {
	Instances []TimeService
}

// Kill makes a random TimeService instance unresponsive.
func (m *TimeServiceManager) Kill() {
	if len(m.Instances) > 0 {
		n := rand.Intn(len(m.Instances))
		close(m.Instances[n].Dead)
		m.Instances = append(m.Instances[:n], m.Instances[n+1:]...)
	}
}

// Spawn creates a new TimeService instance.
func (m *TimeServiceManager) Spawn() TimeService {
	ts := TimeService{
		Dead:            make(chan struct{}, 0),
		ReqChan:         make(chan Request, 10),
		AvgResponseTime: rand.Float64() * 3,
	}
	m.Instances = append(m.Instances, ts)

	return ts
}
