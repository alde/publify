package worker

import (
	"context"
	"runtime"
	"sync"

	"github.com/alde/publify/pkg/progress"
)

// Job represents a unit of work to be processed (because work should be organized, unlike my desk)
type Job interface {
	Process(ctx context.Context) error
	ID() string
}

// Result contains the outcome of processing a job (success or failure, like Swedish weather)
type Result struct {
	JobID string
	Error error
}

// Pool manages a pool of worker goroutines (Swedish teamwork in digital form)
type Pool struct {
	workerCount int
	jobs        chan Job
	results     chan Result
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	progress    *progress.ProgressTracker
}

// NewPool creates a new worker pool (because CPUs need management too, ja?)
func NewPool(workerCount int) *Pool {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU() // When in doubt, use what the machine gives you
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		workerCount: workerCount,
		jobs:        make(chan Job, workerCount*2), // Buffer to prevent blocking
		results:     make(chan Result, workerCount*2),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// NewPoolWithProgress creates a new worker pool with progress tracking
func NewPoolWithProgress(workerCount, totalJobs int) *Pool {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		workerCount: workerCount,
		jobs:        make(chan Job, workerCount*2),
		results:     make(chan Result, workerCount*2),
		ctx:         ctx,
		cancel:      cancel,
		progress:    progress.NewProgressTracker(workerCount, totalJobs),
	}
}

// Start begins processing jobs
func (p *Pool) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop gracefully shuts down the pool
func (p *Pool) Stop() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
	p.cancel()

	if p.progress != nil {
		p.progress.Finish()
	}
}

// ForceStop immediately cancels all work
func (p *Pool) ForceStop() {
	p.cancel()
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// Submit adds a job to the processing queue
func (p *Pool) Submit(job Job) {
	select {
	case p.jobs <- job:
	case <-p.ctx.Done():
		// Pool is shutting down
		p.results <- Result{
			JobID: job.ID(),
			Error: p.ctx.Err(),
		}
	}
}

// Results returns the results channel
func (p *Pool) Results() <-chan Result {
	return p.results
}

// worker processes jobs from the jobs channel
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				return // Channel closed, worker should exit
			}

			// Update progress - starting job
			if p.progress != nil {
				p.progress.UpdateWorker(id, job.ID(), false)
			}

			err := job.Process(p.ctx)

			// Update progress - job completed
			if p.progress != nil {
				p.progress.UpdateWorker(id, job.ID(), true)
			}

			p.results <- Result{
				JobID: job.ID(),
				Error: err,
			}

		case <-p.ctx.Done():
			return // Context cancelled, worker should exit
		}
	}
}

// WorkerCount returns the number of workers in the pool
func (p *Pool) WorkerCount() int {
	return p.workerCount
}
