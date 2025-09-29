package progress

import (
	"fmt"
	"sync"
	"time"
)

// WorkerProgress tracks progress for individual workers
type WorkerProgress struct {
	WorkerID    int
	JobsTotal   int
	JobsCompleted int
	CurrentJob  string
	LastUpdate  time.Time
}

// ProgressTracker manages progress across multiple workers
type ProgressTracker struct {
	mu           sync.RWMutex
	workers      map[int]*WorkerProgress
	totalJobs    int
	completedJobs int
	startTime    time.Time
	lastDisplay  time.Time
	displayRate  time.Duration
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(workerCount, totalJobs int) *ProgressTracker {
	tracker := &ProgressTracker{
		workers:     make(map[int]*WorkerProgress),
		totalJobs:   totalJobs,
		startTime:   time.Now(),
		displayRate: 500 * time.Millisecond, // Update display every 500ms
	}

	// Initialize worker progress
	for i := 0; i < workerCount; i++ {
		tracker.workers[i] = &WorkerProgress{
			WorkerID:   i,
			LastUpdate: time.Now(),
		}
	}

	return tracker
}

// UpdateWorker updates progress for a specific worker
func (pt *ProgressTracker) UpdateWorker(workerID int, jobDescription string, completed bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	worker := pt.workers[workerID]
	if worker == nil {
		return
	}

	worker.CurrentJob = jobDescription
	worker.LastUpdate = time.Now()

	if completed {
		worker.JobsCompleted++
		pt.completedJobs++
	}

	// Display progress if enough time has passed
	if time.Since(pt.lastDisplay) >= pt.displayRate {
		pt.displayProgress()
		pt.lastDisplay = time.Now()
	}
}

// displayProgress shows current progress across all workers
func (pt *ProgressTracker) displayProgress() {
	elapsed := time.Since(pt.startTime)
	percentage := float64(pt.completedJobs) / float64(pt.totalJobs) * 100

	// Estimate time remaining
	var eta time.Duration
	if pt.completedJobs > 0 {
		avgTimePerJob := elapsed / time.Duration(pt.completedJobs)
		remainingJobs := pt.totalJobs - pt.completedJobs
		eta = avgTimePerJob * time.Duration(remainingJobs)
	}

	// Clear previous lines and redraw
	fmt.Print("\033[2K\r") // Clear current line

	// Overall progress
	fmt.Printf("Progress: %d/%d (%.1f%%) | Elapsed: %v | ETA: %v\n",
		pt.completedJobs, pt.totalJobs, percentage,
		elapsed.Round(time.Second), eta.Round(time.Second))

	// Worker details (show active workers)
	activeWorkers := 0
	for _, worker := range pt.workers {
		if worker.CurrentJob != "" {
			activeWorkers++
			status := "ACTIVE"
			if time.Since(worker.LastUpdate) > 2*time.Second {
				status = "STALLED"
			}

			jobDesc := worker.CurrentJob
			if len(jobDesc) > 30 {
				jobDesc = jobDesc[:27] + "..."
			}

			fmt.Printf("  Worker %d [%s] %s (completed: %d)\n",
				worker.WorkerID, status, jobDesc, worker.JobsCompleted)
		}
	}

	if activeWorkers == 0 {
		fmt.Printf("  All workers idle\n")
	}

	// Move cursor up to overwrite on next update
	fmt.Printf("\033[%dA", activeWorkers+2) // Move up (workers + header + 1)
}

// Finish completes the progress tracking and shows final stats
func (pt *ProgressTracker) Finish() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Clear the progress display area
	activeWorkers := len(pt.workers)
	for i := 0; i < activeWorkers+3; i++ {
		fmt.Print("\033[2K\n") // Clear line and move down
	}
	fmt.Printf("\033[%dA", activeWorkers+3) // Move back up

	elapsed := time.Since(pt.startTime)

	fmt.Printf("Completed %d jobs in %v\n",
		pt.completedJobs, elapsed.Round(time.Millisecond))

	// Show final worker stats
	fmt.Printf("Worker Statistics:\n")
	for workerID, worker := range pt.workers {
		rate := float64(worker.JobsCompleted) / elapsed.Seconds()
		fmt.Printf("  Worker %d: %d jobs (%.1f jobs/sec)\n",
			workerID, worker.JobsCompleted, rate)
	}
	fmt.Println()
}

// GetStats returns current progress statistics
func (pt *ProgressTracker) GetStats() ProgressStats {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	elapsed := time.Since(pt.startTime)
	rate := 0.0
	if elapsed.Seconds() > 0 {
		rate = float64(pt.completedJobs) / elapsed.Seconds()
	}

	return ProgressStats{
		TotalJobs:     pt.totalJobs,
		CompletedJobs: pt.completedJobs,
		WorkerCount:   len(pt.workers),
		Elapsed:       elapsed,
		Rate:          rate,
		Percentage:    float64(pt.completedJobs) / float64(pt.totalJobs) * 100,
	}
}

// ProgressStats contains progress statistics
type ProgressStats struct {
	TotalJobs     int
	CompletedJobs int
	WorkerCount   int
	Elapsed       time.Duration
	Rate          float64 // Jobs per second
	Percentage    float64
}

// SimpleProgress provides a basic progress bar for non-worker tasks
type SimpleProgress struct {
	total   int
	current int
	label   string
	width   int
}

// NewSimpleProgress creates a simple progress bar
func NewSimpleProgress(total int, label string) *SimpleProgress {
	return &SimpleProgress{
		total: total,
		label: label,
		width: 40,
	}
}

// Update updates the simple progress bar
func (sp *SimpleProgress) Update(current int) {
	sp.current = current
	sp.display()
}

// display shows the current progress bar
func (sp *SimpleProgress) display() {
	percentage := float64(sp.current) / float64(sp.total) * 100
	filled := int(float64(sp.width) * float64(sp.current) / float64(sp.total))

	bar := ""
	for i := 0; i < sp.width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	fmt.Printf("\r%s [%s] %d/%d (%.1f%%)",
		sp.label, bar, sp.current, sp.total, percentage)
}

// Finish completes the simple progress bar
func (sp *SimpleProgress) Finish() {
	sp.Update(sp.total)
	fmt.Println(" DONE")
}