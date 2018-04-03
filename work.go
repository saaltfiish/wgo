// Package utils provides job queue
// reference: http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/
package wgo

import (
	"time"
)

type Job struct {
	Payload interface{}      `json:"payload,omitempty"`
	Result  chan interface{} `json:"result,omitempty"`
}

type JobFunc func(interface{}) interface{}

type JobWorker struct {
	pool    chan chan *Job
	channel chan *Job
	quit    chan bool
	handler JobFunc
}

type WorkerPool struct {
	// A pool of workers channels that are registered with the dispatcher
	name    string
	queue   chan *Job
	pool    chan chan *Job
	max     int
	handler JobFunc
	workers []*JobWorker
}

// new job
func newJob(pl interface{}) *Job {
	return &Job{
		Payload: pl,
		Result:  make(chan interface{}, 1),
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
				// we have received a job
				job.Result <- jw.handler(job.Payload)
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
func NewWorkerPool(name string, maxWorkers int, jf JobFunc) *WorkerPool {
	pool := make(chan chan *Job, maxWorkers)
	queue := make(chan *Job)
	return &WorkerPool{
		name:    name,
		queue:   queue,
		pool:    pool,
		max:     maxWorkers,
		handler: jf,
		workers: make([]*JobWorker, 0),
	}
}

// register to default var
func (workerPool *WorkerPool) Register() {
	wp = workerPool
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
func Push(i interface{}) {
	if wp != nil {
		wp.push(i)
	}
}

// req job, 同步
func Req(i interface{}) {
	if wp != nil {
		wp.req(i)
	}
}

func PushTo(name string, i interface{}) {
	for _, work := range wgo.works {
		if work.Name() == name {
			work.push(i)
		}
	}
}
func (work *WorkerPool) push(i interface{}) {
	work.queue <- newJob(i)
}
func (work *WorkerPool) req(i interface{}) {
	job := newJob(i)
	work.queue <- job
	// waiting result, timeout in 10 seconds
	to := time.Tick(10 * time.Second)
	select {
	case <-to: //超时
		Info("timeout in 10s")
	case result := <-job.Result:
		Info("received result: %+v", result)
	}
}
