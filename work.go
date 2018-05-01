// Package utils provides job queue
// reference: http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/
package wgo

import (
	"fmt"
	"time"

	"wgo/utils"
)

type Job struct {
	id      string
	method  string
	payload interface{}
	result  chan interface{}
}

func (j *Job) ID() string {
	return j.id
}
func (j *Job) Method() string {
	return j.method
}
func (j *Job) Payload() interface{} {
	return j.payload
}

type JobHandler func(*Job) interface{}

type JobWorker struct {
	pool    chan chan *Job
	channel chan *Job
	quit    chan bool
	handler JobHandler
	routes  map[string]*JobRoute
}

type WorkerPool struct {
	// A pool of workers channels that are registered with the dispatcher
	name    string
	queue   chan *Job
	pool    chan chan *Job
	max     int
	handler JobHandler // default handler
	routes  map[string]*JobRoute
	workers []*JobWorker
}

type JobRoute struct {
	method  string
	handler JobHandler
}

// new job
func NewJob(pl interface{}, opts ...interface{}) *Job {
	method := ""
	id := ""
	if len(opts) > 0 {
		if m, ok := opts[0].(string); ok && m != "" {
			method = m
		}
	}
	if len(opts) > 1 {
		if r, ok := opts[1].(string); ok && r != "" {
			id = r
		}
	}
	// generate random job id
	if id == "" {
		id = utils.FastRequestId(16)
	}
	return &Job{
		id:      id,
		method:  method,
		payload: pl,
		result:  make(chan interface{}, 1),
	}
}

// worker run
func (jw *JobWorker) Run(sn int) {
	go func() {
		for {
			// register the current worker into the worker queue.
			jw.pool <- jw.channel

			select {
			case job := <-jw.channel:
				// we have received a job, route it
				handler := jw.handler // default handler
				if job.method != "" {
					if route, ok := jw.routes[job.method]; ok {
						handler = route.handler
					}
				}
				job.result <- handler(job)
			case <-jw.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}

// job worker stop
func (jw *JobWorker) Stop() {
	go func() {
		jw.quit <- true
	}()
}

// create new worker pool
func NewWorkerPool(name string, maxWorkers int, handler JobHandler) *WorkerPool {
	pool := make(chan chan *Job, maxWorkers)
	queue := make(chan *Job)
	return &WorkerPool{
		name:    name,
		queue:   queue,
		pool:    pool,
		max:     maxWorkers,
		handler: handler,
		routes:  make(map[string]*JobRoute),
		workers: make([]*JobWorker, 0),
	}
}

// register to default var
func (workerPool *WorkerPool) Register() {
	wp = workerPool
}

// routes for methods
func (workerPool *WorkerPool) Add(method string, handler JobHandler) *WorkerPool {
	workerPool.routes[method] = &JobRoute{
		method:  method,
		handler: handler,
	}
	return workerPool
}

// worker poll name
func (wp *WorkerPool) Name() string {
	return wp.name
}

// pool start
func (wp *WorkerPool) Start() *WorkerPool {
	// starting n number of workers
	for i := 0; i < wp.max; i++ {
		worker := &JobWorker{
			pool:    wp.pool,
			channel: make(chan *Job),
			quit:    make(chan bool),
			handler: wp.handler,
			routes:  wp.routes,
		}
		worker.Run(i)
		wp.workers = append(wp.workers, worker)
	}

	go wp.dispatch()

	return wp
}

// pool end
func (wp *WorkerPool) End() {
	for _, worker := range wp.workers {
		worker.Stop()
	}
}

// pool dispatch
func (wp *WorkerPool) dispatch() {
	for {
		select {
		case job := <-wp.queue:
			// a job request has been received
			go func(job *Job) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				channel := <-wp.pool

				// dispatch the job to the worker job channel
				channel <- job
			}(job)
		}
	}
}

// push job, 异步
func Push(i interface{}, opts ...interface{}) {
	if wp != nil {
		wp.push(i, opts...)
	}
}

// req job, 同步
func Req(i interface{}, opts ...interface{}) (interface{}, error) {
	if wp != nil {
		return wp.req(i, opts...)
	}
	return nil, fmt.Errorf("not found worker pool")
}

func PushTo(name string, i interface{}, opts ...interface{}) {
	for _, work := range wgo.works {
		if work.Name() == name {
			work.push(i, opts...)
		}
	}
}

func ReqTo(name string, i interface{}, opts ...interface{}) (interface{}, error) {
	for _, work := range wgo.works {
		if work.Name() == name {
			return work.req(i, opts...)
		}
	}
	return nil, fmt.Errorf("not found worker pool")
}

func (work *WorkerPool) push(i interface{}, opts ...interface{}) {
	if job, ok := i.(*Job); ok {
		// 如果直接传入job, 欣然接受, 忽略opts
		work.queue <- job
	} else {
		// 封装为job, method为空, 这样默认handler会处理这个job
		work.queue <- NewJob(i, opts...)
	}
}
func (work *WorkerPool) req(i interface{}, opts ...interface{}) (interface{}, error) {
	var job *Job
	if ijob, ok := i.(*Job); ok {
		// 如果直接传入job, 欣然接受, 忽略opts
		job = ijob
	} else {
		// 封装为job, method为空, 这样默认handler会处理这个job
		job = NewJob(i, opts...)
	}
	work.queue <- job
	// waiting result, timeout in 10 seconds
	to := time.Tick(10 * time.Second)
	select {
	case <-to: //超时
		Warn("timeout in 10s")
		return nil, fmt.Errorf("timeout in 10s")
	case result := <-job.result:
		// Info("received job result: %+v", result)
		if err, ok := result.(error); ok {
			return nil, err
		} else {
			return result, nil
		}
	}
}
