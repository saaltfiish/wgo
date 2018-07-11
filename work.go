// Package utils provides job queue
// reference: http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/
package wgo

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"wgo/server"
	"wgo/utils"
)

type Job struct {
	id       string
	context  *Context // 异步job直接使用Context会有问题, todo: 应该clone一个context
	work     string
	method   string
	payload  interface{}
	result   interface{}
	req      []interface{} // 对外的请求
	resp     []interface{} // 对外请求的返回
	err      *server.ServerError
	response chan interface{}
}

func (j *Job) ID() string {
	return j.id
}
func (j *Job) Context() *Context {
	return j.context
}
func (j *Job) Work() string {
	return j.work
}
func (j *Job) Method() string {
	return j.method
}
func (j *Job) Payload() interface{} {
	return j.payload
}
func (j *Job) Error(err error) {
	j.err = server.WrapError(err)
}
func (j *Job) Result() interface{} {
	return j.result
}
func (j *Job) SetResult(i interface{}) error {
	j.result = i
	return nil
}
func (j *Job) Response() {
	j.SaveAccessLog()
	if j.err != nil {
		j.response <- j.err
	} else {
		j.response <- j.result
	}
}
func (j *Job) SaveReq(i interface{}) {
	j.req = append(j.req, i)
}
func (j *Job) SaveResp(i interface{}) {
	j.resp = append(j.resp, i)
}
func (j *Job) SaveAccessLog() {
	c := j.Context()
	ac := c.Access().Clone()
	// user info
	ac.Service.User.Id = c.UserID()
	ac.Service.Endpoint = j.Work()
	ac.Service.Desc = j.Method()
	ac.Service.Action = "C"
	// new & old, new对应对外请求, old对应对外请求的返回
	if j.req != nil {
		if nb, err := json.Marshal(j.req); err == nil {
			ac.Service.New = string(nb)
		}
	}
	if j.resp != nil {
		if nb, err := json.Marshal(j.resp); err == nil {
			ac.Service.Old = string(nb)
		}
	}
	if sa, err := json.Marshal(ac); err != nil {
		c.Error("serialize access data failed: %s", err)
	} else {
		Accessor().Access(string(sa))
	}
}

// 继续处理job, 上一阶段的result作为下一阶段的payload
func (j *Job) Continue() {
	j.payload = j.result
	j.result = nil
}

func (c *Context) NewJob(name, method string, pl interface{}, opts ...interface{}) *Job {
	id := c.RequestID()
	// generate random job id
	if id == "" {
		id = utils.FastRequestId(16)
	}
	c.job = &Job{
		id:       id,
		context:  c,
		work:     name,
		method:   method,
		payload:  pl,
		response: make(chan interface{}, 1),
		req:      make([]interface{}, 0),
		resp:     make([]interface{}, 0),
	}
	return c.job
}

// type JobHandler func(*Job) interface{}
//
// func (jh JobHandler) Do(job *Job) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			var err error
// 			switch r := r.(type) {
// 			case error:
// 				err = r
// 			default:
// 				err = fmt.Errorf("%v", r)
// 			}
// 			stack := make([]byte, 64<<10)
// 			length := runtime.Stack(stack, false)
// 			Error("[wgo.work] %s %s", err, stack[:length])
// 		}
// 	}()
// 	res := jh(job)
// 	job.result <- res
// }

type JobWorker struct {
	pool    chan chan *Job
	channel chan *Job
	quit    chan bool
	handler HandlerFunc
	routes  map[string]*JobRoute
}

type WorkerPool struct {
	// A pool of workers channels that are registered with the dispatcher
	name    string
	queue   chan *Job
	pool    chan chan *Job
	max     int
	handler HandlerFunc // default handler
	routes  map[string]*JobRoute
	workers []*JobWorker
}

type JobRoute struct {
	method      string
	middlewares []MiddlewareFunc
	handlers    []HandlerFunc
}

// 使用中间件
func (jr *JobRoute) Use(m MiddlewareFunc) *JobRoute {
	if jr.middlewares == nil {
		jr.middlewares = make([]MiddlewareFunc, 0)
	}
	jr.middlewares = append(jr.middlewares, m)
	return jr
}

func (hf HandlerFunc) Do(j *Job) {
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch r := r.(type) {
			case error:
				err = r
			default:
				err = fmt.Errorf("%v", r)
			}
			stack := make([]byte, 64<<10)
			length := runtime.Stack(stack, false)
			Error("[serve.job] %s %s", err, stack[:length])
			j.Error(err)
			j.Response()
		}
	}()
	c := j.Context()
	err := hf(c)
	if err != nil {
		c.Error("serve job error: %s", err)
		j.Error(err)
	}
	j.Response()
}

// 按顺序链式执行
func (this HandlerFunc) Chain(next HandlerFunc) HandlerFunc {
	return func(c *Context) error {
		if err := this(c); err != nil {
			return err
		}
		// access logging, 每一步都记录access log
		c.Job().SaveAccessLog()
		// 以上一个result作为下一个payload
		c.Job().Continue()
		return next(c)
	}
}

func jobHandler(method string, route *JobRoute) HandlerFunc {
	handler := route.handlers[0]
	for _, h := range route.handlers[1:] {
		handler = handler.Chain(h)
	}
	if len(route.middlewares) > 0 {
		for i := len(route.middlewares) - 1; i >= 0; i-- {
			handler = route.middlewares[i](handler)
		}
	}
	return handler
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
					if route, ok := jw.routes[job.method]; ok && len(route.handlers) > 0 {
						// handler = route.handlers[0]
						// for _, h := range route.handlers[1:] {
						// 	handler = handler.Chain(h)
						// }
						// if len(route.middlewares) > 0 {
						// 	for i := len(route.middlewares) - 1; i >= 0; i-- {
						// 		handler = route.middlewares[i](handler)
						// 	}
						// }
						handler = jobHandler(job.method, route)
					}
				}
				handler.Do(job)
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
func NewWorkerPool(name string, maxWorkers int, handler HandlerFunc) *WorkerPool {
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
func (workerPool *WorkerPool) Add(method string, handlers ...HandlerFunc) *JobRoute {
	workerPool.routes[method] = &JobRoute{
		method:   method,
		handlers: handlers,
	}
	return workerPool.routes[method]
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
func (c *Context) Push(method string, i interface{}, opts ...interface{}) {
	if wp != nil {
		wp.push(c, method, i, opts...)
	}
}

// req job, 同步
func (c *Context) Req(method string, i interface{}, opts ...interface{}) (interface{}, error) {
	if wp != nil {
		return wp.req(c, method, i, opts...)
	}
	return nil, fmt.Errorf("not found worker pool")
}

func (c *Context) PushTo(name string, method string, i interface{}, opts ...interface{}) {
	for _, work := range wgo.works {
		if work.Name() == name {
			work.push(c, method, i, opts...)
		}
	}
}

func (c *Context) ReqTo(name string, method string, i interface{}, opts ...interface{}) (interface{}, error) {
	for _, work := range wgo.works {
		if work.Name() == name {
			return work.req(c, method, i, opts...)
		}
	}
	return nil, fmt.Errorf("not found worker pool")
}

func (work *WorkerPool) push(c *Context, method string, i interface{}, opts ...interface{}) {
	// 封装为job, method为空, 这样默认handler会处理这个job
	work.queue <- c.NewJob(work.Name(), method, i, opts...)
}
func (work *WorkerPool) req(c *Context, method string, i interface{}, opts ...interface{}) (interface{}, error) {
	// 封装为job, method为空, 这样默认handler会处理这个job
	job := c.NewJob(work.Name(), method, i, opts...)
	work.queue <- job
	// waiting result, timeout in 60 seconds
	to := time.Tick(60 * time.Second)
	select {
	case <-to: //超时
		Warn("timeout in 60s")
		return nil, fmt.Errorf("timeout in 60s")
	case response := <-job.response:
		// Info("received job response: %+v", response)
		if err, ok := response.(error); ok {
			return nil, err
		} else {
			return response, nil
		}
	}
}
