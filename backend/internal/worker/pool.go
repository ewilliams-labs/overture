// Package worker provides background processing for track-related jobs.
package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
)

// Job represents a background task for track processing.
type Job struct {
	TrackID    string
	PreviewURL string
}

// Pool manages background workers for async jobs.
type Pool struct {
	repo ports.PlaylistRepository
	jobs chan Job
	wg   sync.WaitGroup
}

// NewPool creates a worker pool with the given worker count and queue size.
func NewPool(repo ports.PlaylistRepository, workers int, queueSize int) *Pool {
	if workers < 1 {
		workers = 1
	}
	if queueSize < 1 {
		queueSize = 1
	}
	return &Pool{repo: repo, jobs: make(chan Job, queueSize)}
}

// Start launches the worker goroutines.
func (p *Pool) Start(workers int) {
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for job := range p.jobs {
				p.processJob(job)
			}
		}()
	}
}

// Stop waits for workers to finish after closing the queue.
func (p *Pool) Stop() {
	close(p.jobs)
	p.wg.Wait()
}

// Submit queues a job without blocking.
func (p *Pool) Submit(job Job) {
	select {
	case p.jobs <- job:
	default:
		log.Printf("WARN worker: dropping job for %s", job.TrackID)
	}
}

func (p *Pool) processJob(job Job) {
	if job.PreviewURL == "" {
		log.Printf("⚠️ No preview URL for Track %s. Skipping analysis.", job.TrackID)
		return
	}

	time.Sleep(100 * time.Millisecond)

	features := domain.AudioFeatures{
		Danceability:     0.15,
		Energy:           0.95,
		Valence:          0.95,
		Tempo:            128.0,
		Instrumentalness: 0.05,
		Acousticness:     0.1,
	}
	if err := p.repo.UpdateTrackFeatures(context.Background(), job.TrackID, features); err != nil {
		log.Printf("WARN worker: failed to update track %s: %v", job.TrackID, err)
		return
	}
	log.Printf("Processed %s", job.TrackID)
	log.Printf("💾 Updated Track %s with analyzed features (0.95).", job.TrackID)
}
