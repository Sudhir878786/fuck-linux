package detector

import (
	"math"
	"testing"
	"time"
)

func TestMagnitude(t *testing.T) {
	got := magnitude(3, 4, 0)
	if math.Abs(got-5.0) > 1e-9 {
		t.Errorf("magnitude(3,4,0) = %f, want 5.0", got)
	}
}

func TestClassifySeverity(t *testing.T) {
	tests := []struct {
		amp  float64
		want Severity
	}{
		{0.01, SeverityLight},
		{0.14, SeverityLight},
		{0.15, SeverityMedium},
		{0.49, SeverityMedium},
		{0.50, SeverityHard},
		{1.00, SeverityHard},
	}
	for _, tt := range tests {
		if got := classifySeverity(tt.amp); got != tt.want {
			t.Errorf("classifySeverity(%f) = %s, want %s", tt.amp, got, tt.want)
		}
	}
}

func TestDetectorBasicSpike(t *testing.T) {
	cfg := Config{MinAmplitude: 0.1, Cooldown: 100 * time.Millisecond}
	d := New(cfg)
	now := time.Now()

	// Feed a stable baseline (gravity ~9.8 on Z)
	for i := 0; i < ringSize; i++ {
		d.Process(0, 0, 9.8, now.Add(time.Duration(i)*time.Millisecond))
	}

	// Spike: sudden large value
	ev := d.Process(0, 0, 12.0, now.Add(time.Duration(ringSize)*time.Millisecond))
	if ev == nil {
		t.Fatal("expected event from spike, got nil")
	}
	if ev.Amplitude < 0.1 {
		t.Errorf("amplitude %f below threshold", ev.Amplitude)
	}
}

func TestDetectorCooldown(t *testing.T) {
	cfg := Config{MinAmplitude: 0.1, Cooldown: 500 * time.Millisecond}
	d := New(cfg)
	now := time.Now()

	// Build baseline
	for i := 0; i < ringSize; i++ {
		d.Process(0, 0, 9.8, now.Add(time.Duration(i)*time.Millisecond))
	}

	base := now.Add(time.Duration(ringSize) * time.Millisecond)

	// First spike triggers event
	ev1 := d.Process(0, 0, 12.0, base)
	if ev1 == nil {
		t.Fatal("expected first event")
	}

	// Second spike within cooldown should NOT trigger
	ev2 := d.Process(0, 0, 12.0, base.Add(100*time.Millisecond))
	if ev2 != nil {
		t.Error("expected nil event during cooldown")
	}

	// Third spike after cooldown SHOULD trigger
	// Need to rebuild baseline a bit
	for i := 0; i < ringSize; i++ {
		d.Process(0, 0, 9.8, base.Add(200*time.Millisecond+time.Duration(i)*time.Millisecond))
	}
	ev3 := d.Process(0, 0, 12.0, base.Add(600*time.Millisecond))
	if ev3 == nil {
		t.Error("expected event after cooldown")
	}
}

func TestDrainEvents(t *testing.T) {
	cfg := Config{MinAmplitude: 0.1, Cooldown: 10 * time.Millisecond}
	d := New(cfg)
	now := time.Now()

	for i := 0; i < ringSize; i++ {
		d.Process(0, 0, 9.8, now.Add(time.Duration(i)*time.Millisecond))
	}

	d.Process(0, 0, 12.0, now.Add(time.Duration(ringSize)*time.Millisecond))

	events := d.DrainEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	events = d.DrainEvents()
	if len(events) != 0 {
		t.Fatalf("expected 0 events after drain, got %d", len(events))
	}
}
