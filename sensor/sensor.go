package sensor

import "time"

// Sample represents a single 3-axis accelerometer reading.
type Sample struct {
	X, Y, Z float64
	Time     time.Time
}

// Sensor is the pluggable interface for all accelerometer/impact sources.
// Implementations must be safe to call from a single goroutine.
type Sensor interface {
	// Start begins reading sensor data. It blocks until ctx is cancelled
	// or an error occurs. Samples are sent on the provided channel.
	Start(samples chan<- Sample) error

	// Close releases any resources held by the sensor.
	Close() error

	// Name returns a human-readable sensor name.
	Name() string
}
