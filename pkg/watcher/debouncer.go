package watcher

import (
	"context"
	"github.com/ritzau/deps-analyzer/pkg/logging"
	"sync"
	"time"
)

// Debouncer batches rapid file system events to avoid excessive re-analysis
type Debouncer struct {
	input       <-chan ChangeEvent
	output      chan ChangeEvent
	quietPeriod time.Duration
	maxWait     time.Duration
	mu          sync.Mutex
}

// NewDebouncer creates a new event debouncer
func NewDebouncer(input <-chan ChangeEvent, quietPeriod, maxWait time.Duration) *Debouncer {
	return &Debouncer{
		input:       input,
		output:      make(chan ChangeEvent, 10),
		quietPeriod: quietPeriod,
		maxWait:     maxWait,
	}
}

// Start begins processing events with debouncing
func (d *Debouncer) Start(ctx context.Context) {
	go d.run(ctx)
}

// run processes events and applies debouncing logic
func (d *Debouncer) run(ctx context.Context) {
	var (
		timer        *time.Timer
		maxWaitTimer *time.Timer
		accumulated  = make(map[ChangeType][]string)
		eventCount   int
	)

	flush := func() {
		if eventCount == 0 {
			return
		}

		logging.Debug("flushing accumulated events", "count", eventCount)

		// Send events in order: BUILD files first (need full analysis), then others
		if paths, ok := accumulated[ChangeTypeBuildFile]; ok && len(paths) > 0 {
			d.output <- ChangeEvent{
				Type:      ChangeTypeBuildFile,
				Paths:     paths,
				Timestamp: time.Now(),
			}
		}
		if paths, ok := accumulated[ChangeTypeDFile]; ok && len(paths) > 0 {
			d.output <- ChangeEvent{
				Type:      ChangeTypeDFile,
				Paths:     paths,
				Timestamp: time.Now(),
			}
		}
		if paths, ok := accumulated[ChangeTypeOFile]; ok && len(paths) > 0 {
			d.output <- ChangeEvent{
				Type:      ChangeTypeOFile,
				Paths:     paths,
				Timestamp: time.Now(),
			}
		}

		// Reset accumulators
		accumulated = make(map[ChangeType][]string)
		eventCount = 0

		// Stop timers
		if timer != nil {
			timer.Stop()
		}
		if maxWaitTimer != nil {
			maxWaitTimer.Stop()
		}
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			close(d.output)
			return

		case event, ok := <-d.input:
			if !ok {
				flush()
				close(d.output)
				return
			}

			// Accumulate event
			accumulated[event.Type] = append(accumulated[event.Type], event.Paths...)
			eventCount++

			// Reset quiet period timer
			if timer == nil {
				timer = time.AfterFunc(d.quietPeriod, flush)
			} else {
				timer.Reset(d.quietPeriod)
			}

			// Start max wait timer on first event
			if maxWaitTimer == nil {
				maxWaitTimer = time.AfterFunc(d.maxWait, flush)
			}

		case <-func() <-chan time.Time {
			if timer != nil {
				return timer.C
			}
			return nil
		}():
			flush()

		case <-func() <-chan time.Time {
			if maxWaitTimer != nil {
				return maxWaitTimer.C
			}
			return nil
		}():
			flush()
		}
	}
}

// Output returns the channel of debounced events
func (d *Debouncer) Output() <-chan ChangeEvent {
	return d.output
}
