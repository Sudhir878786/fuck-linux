package detector

import (
	"math"
	"sync"
	"time"
)

// Severity classifies the intensity of a detected slap.
type Severity string

const (
	SeverityLight  Severity = "light"
	SeverityMedium Severity = "medium"
	SeverityHard   Severity = "hard"
)

// Event represents a detected slap/impact.
type Event struct {
	Time      time.Time
	Amplitude float64
	Severity  Severity
}

// Config holds tuning parameters for the detector.
type Config struct {
	// MinAmplitude is the minimum acceleration magnitude delta to register.
	MinAmplitude float64

	// Cooldown is the minimum time between consecutive events.
	Cooldown time.Duration
}

// DefaultConfig returns sensible detection defaults.
func DefaultConfig() Config {
	return Config{
		MinAmplitude: 0.05,
		Cooldown:     750 * time.Millisecond,
	}
}

// Detector processes a stream of accelerometer samples and emits events
// when a slap/impact is detected. It uses a simple high-pass filter
// and spike detection algorithm.
type Detector struct {
	mu     sync.Mutex
	config Config

	// Ring buffer of recent magnitudes for baseline tracking
	ring     [ringSize]float64
	ringIdx  int
	ringFull bool

	// Baseline (moving average of the ring)
	baseline float64

	// Last event time (for cooldown)
	lastEvent time.Time

	// Events accumulates detected slap events. Caller should drain it.
	Events []Event
}

const ringSize = 64

// New creates a detector with the given config.
func New(cfg Config) *Detector {
	return &Detector{config: cfg}
}

// magnitude returns the Euclidean norm of a 3D vector.
func magnitude(x, y, z float64) float64 {
	return math.Sqrt(x*x + y*y + z*z)
}

// classifySeverity maps amplitude to a severity level.
func classifySeverity(amplitude float64) Severity {
	switch {
	case amplitude >= 0.5:
		return SeverityHard
	case amplitude >= 0.15:
		return SeverityMedium
	default:
		return SeverityLight
	}
}

// Process ingests a single accelerometer sample and checks for spikes.
// Returns a non-nil Event if a slap was detected.
func (d *Detector) Process(x, y, z float64, t time.Time) *Event {
	d.mu.Lock()
	defer d.mu.Unlock()

	mag := magnitude(x, y, z)

	// Update the ring buffer and compute the moving average baseline.
	d.ring[d.ringIdx] = mag
	d.ringIdx = (d.ringIdx + 1) % ringSize
	if d.ringIdx == 0 {
		d.ringFull = true
	}

	count := ringSize
	if !d.ringFull {
		count = d.ringIdx
	}
	if count == 0 {
		// First sample, just record baseline
		d.baseline = mag
		return nil
	}

	var sum float64
	for i := 0; i < count; i++ {
		sum += d.ring[i]
	}
	d.baseline = sum / float64(count)

	// The spike amplitude is how much this sample exceeds the baseline.
	amplitude := mag - d.baseline
	if amplitude < d.config.MinAmplitude {
		return nil
	}

	// Cooldown check
	if !d.lastEvent.IsZero() && t.Sub(d.lastEvent) < d.config.Cooldown {
		return nil
	}

	d.lastEvent = t
	ev := &Event{
		Time:      t,
		Amplitude: amplitude,
		Severity:  classifySeverity(amplitude),
	}
	d.Events = append(d.Events, *ev)
	return ev
}

// DrainEvents returns all accumulated events and clears the buffer.
func (d *Detector) DrainEvents() []Event {
	d.mu.Lock()
	defer d.mu.Unlock()
	events := d.Events
	d.Events = nil
	return events
}

// UpdateConfig replaces the detector configuration atomically.
func (d *Detector) UpdateConfig(cfg Config) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.config = cfg
}
