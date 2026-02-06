package api

import (
	"sync"
	"time"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusSuccess    JobStatus = "success"
	StatusFailed     JobStatus = "failed"
)

type Job struct {
	ID          string    `json:"id"`
	Status      JobStatus `json:"status"`
	Progress    int       `json:"progress"` // 0-100
	Message     string    `json:"message"`
	DownloadURL string    `json:"download_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Error       string    `json:"error,omitempty"`
}

type JobManager struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

var GlobalJobManager = &JobManager{
	jobs: make(map[string]*Job),
}

func (jm *JobManager) CreateJob(id string) *Job {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	job := &Job{
		ID:        id,
		Status:    StatusPending,
		Progress:  0,
		Message:   "Queued",
		CreatedAt: time.Now(),
	}
	jm.jobs[id] = job
	return job
}

func (jm *JobManager) GetJob(id string) (*Job, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	job, ok := jm.jobs[id]
	return job, ok
}

func (jm *JobManager) GetAllJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	var list []*Job
	// Sort? Map iteration is random.
	// For now just return list. Frontend can sort.
	for _, job := range jm.jobs {
		list = append(list, job)
	}
	return list
}

func (jm *JobManager) UpdateProgress(id string, progress int, message string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	if job, ok := jm.jobs[id]; ok {
		job.Status = StatusProcessing
		job.Progress = progress
		job.Message = message
	}
}

func (jm *JobManager) CompleteJob(id string, downloadURL string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	if job, ok := jm.jobs[id]; ok {
		job.Status = StatusSuccess
		job.Progress = 100
		job.Message = "Completed"
		job.DownloadURL = downloadURL
	}
}

func (jm *JobManager) FailJob(id string, errStr string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	if job, ok := jm.jobs[id]; ok {
		job.Status = StatusFailed
		job.Message = "Failed"
		job.Error = errStr
	}
}
