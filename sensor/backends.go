package sensor

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	iioBasePath     = "/sys/bus/iio/devices"
	iioReadInterval = 10 * time.Millisecond
)

// IIOSensor reads accelerometer data from the Linux IIO subsystem.
// It looks for iio:deviceN directories with in_accel_x/y/z_raw sysfs files.
type IIOSensor struct {
	devicePath string
	scaleX     float64
	scaleY     float64
	scaleZ     float64
	stop       chan struct{}
}

// NewIIOSensor creates an IIO sensor reader. If devicePath is empty, it
// auto-detects the first accelerometer under /sys/bus/iio/devices.
func NewIIOSensor(devicePath string) (*IIOSensor, error) {
	if devicePath == "" {
		var err error
		devicePath, err = findIIOAccel()
		if err != nil {
			return nil, err
		}
	}

	s := &IIOSensor{
		devicePath: devicePath,
		stop:       make(chan struct{}),
	}

	// Read scale factors (convert raw ADC values to m/s²)
	s.scaleX = readScaleOr(devicePath, "in_accel_x_scale", 1.0)
	s.scaleY = readScaleOr(devicePath, "in_accel_y_scale", 1.0)
	s.scaleZ = readScaleOr(devicePath, "in_accel_z_scale", 1.0)

	// Verify raw files exist
	for _, axis := range []string{"in_accel_x_raw", "in_accel_y_raw", "in_accel_z_raw"} {
		p := filepath.Join(devicePath, axis)
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("IIO axis file missing: %s: %w", p, err)
		}
	}

	return s, nil
}

func (s *IIOSensor) Name() string {
	return fmt.Sprintf("iio:%s", filepath.Base(s.devicePath))
}

func (s *IIOSensor) Start(samples chan<- Sample) error {
	ticker := time.NewTicker(iioReadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return nil
		case <-ticker.C:
			x, err := readAxisRaw(s.devicePath, "in_accel_x_raw")
			if err != nil {
				continue
			}
			y, err := readAxisRaw(s.devicePath, "in_accel_y_raw")
			if err != nil {
				continue
			}
			z, err := readAxisRaw(s.devicePath, "in_accel_z_raw")
			if err != nil {
				continue
			}

			samples <- Sample{
				X:    x * s.scaleX,
				Y:    y * s.scaleY,
				Z:    z * s.scaleZ,
				Time: time.Now(),
			}
		}
	}
}

func (s *IIOSensor) Close() error {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	return nil
}

// findIIOAccel scans /sys/bus/iio/devices for a device with accel channels.
func findIIOAccel() (string, error) {
	entries, err := os.ReadDir(iioBasePath)
	if err != nil {
		return "", fmt.Errorf("cannot read IIO devices at %s: %w", iioBasePath, err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		devPath := filepath.Join(iioBasePath, e.Name())
		// Check if this device has accelerometer channels
		xRaw := filepath.Join(devPath, "in_accel_x_raw")
		if _, err := os.Stat(xRaw); err == nil {
			return devPath, nil
		}
	}

	return "", fmt.Errorf("no IIO accelerometer found under %s", iioBasePath)
}

func readAxisRaw(devPath, filename string) (float64, error) {
	data, err := os.ReadFile(filepath.Join(devPath, filename))
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
}

func readScaleOr(devPath, filename string, fallback float64) float64 {
	data, err := os.ReadFile(filepath.Join(devPath, filename))
	if err != nil {
		// Try the shared scale file
		sharedScale := filepath.Join(devPath, "in_accel_scale")
		data, err = os.ReadFile(sharedScale)
		if err != nil {
			return fallback
		}
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return fallback
	}
	return v
}

// IIOProxySensor reads from iio-sensor-proxy D-Bus service via the
// sysfs compatibility interface at /sys/bus/iio. Falls back to polling
// the accelerometer sysfs files exposed by iio-sensor-proxy.
type IIOProxySensor struct {
	*IIOSensor
}

// NewIIOProxySensor attempts to use iio-sensor-proxy's exposed sysfs path.
// In practice, iio-sensor-proxy exposes the same /sys/bus/iio paths, so
// this is essentially the same as IIOSensor but with a different name
// for clarity.
func NewIIOProxySensor(devicePath string) (*IIOProxySensor, error) {
	inner, err := NewIIOSensor(devicePath)
	if err != nil {
		return nil, fmt.Errorf("iio-sensor-proxy: %w", err)
	}
	return &IIOProxySensor{IIOSensor: inner}, nil
}

func (s *IIOProxySensor) Name() string {
	return "iio-sensor-proxy"
}

// SerialSensor reads newline-separated "X,Y,Z" float values from a
// serial port or file (e.g., /dev/ttyUSB0 for Arduino input).
type SerialSensor struct {
	path string
	file *os.File
	stop chan struct{}
}

// NewSerialSensor opens the given serial device path.
// The device should produce newline-separated lines of "X,Y,Z" floats.
func NewSerialSensor(devicePath string) (*SerialSensor, error) {
	if devicePath == "" {
		devicePath = "/dev/ttyUSB0"
	}

	f, err := os.Open(devicePath)
	if err != nil {
		return nil, fmt.Errorf("opening serial device %s: %w", devicePath, err)
	}

	return &SerialSensor{
		path: devicePath,
		file: f,
		stop: make(chan struct{}),
	}, nil
}

func (s *SerialSensor) Name() string {
	return fmt.Sprintf("serial:%s", s.path)
}

func (s *SerialSensor) Start(samples chan<- Sample) error {
	scanner := bufio.NewScanner(s.file)
	for scanner.Scan() {
		select {
		case <-s.stop:
			return nil
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			continue
		}
		y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			continue
		}
		z, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			continue
		}

		samples <- Sample{
			X:    x,
			Y:    y,
			Z:    z,
			Time: time.Now(),
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case <-s.stop:
			return nil
		default:
			return fmt.Errorf("serial read error: %w", err)
		}
	}
	return nil
}

func (s *SerialSensor) Close() error {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	return s.file.Close()
}

// MicSensor detects physical impacts using microphone input amplitude.
// It reads raw PCM data from an ALSA capture device via arecord and
// computes the RMS energy per window. High-energy bursts map to
// synthetic accelerometer spikes.
type MicSensor struct {
	device     string
	stop       chan struct{}
	sampleRate int
	windowSize int // samples per analysis window
}

// NewMicSensor creates a microphone-based impact sensor.
// device is the ALSA device name (e.g., "default", "hw:0,0").
func NewMicSensor(device string) (*MicSensor, error) {
	if device == "" {
		device = "default"
	}
	return &MicSensor{
		device:     device,
		stop:       make(chan struct{}),
		sampleRate: 44100,
		windowSize: 1024,
	}, nil
}

func (s *MicSensor) Name() string {
	return fmt.Sprintf("mic:%s", s.device)
}

func (s *MicSensor) Start(samples chan<- Sample) error {
	// Use arecord to capture raw PCM S16_LE mono
	// This avoids needing CGo ALSA bindings.
	cmd := execCommand("arecord",
		"-D", s.device,
		"-f", "S16_LE",
		"-r", strconv.Itoa(s.sampleRate),
		"-c", "1",
		"-t", "raw",
		"-q",
		"-")

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mic: stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mic: arecord start failed (is alsa-utils installed?): %w", err)
	}

	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	buf := make([]byte, s.windowSize*2) // 2 bytes per S16_LE sample
	for {
		select {
		case <-s.stop:
			return nil
		default:
		}

		n, err := pipe.Read(buf)
		if err != nil {
			select {
			case <-s.stop:
				return nil
			default:
				return fmt.Errorf("mic: read error: %w", err)
			}
		}

		if n < 4 {
			continue
		}

		// Compute RMS of the PCM window
		numSamples := n / 2
		var sumSq float64
		for i := 0; i < numSamples; i++ {
			// Little-endian signed 16-bit
			sample := int16(uint16(buf[i*2]) | uint16(buf[i*2+1])<<8)
			norm := float64(sample) / 32768.0
			sumSq += norm * norm
		}
		rms := math.Sqrt(sumSq / float64(numSamples))

		// Map RMS to a synthetic Z-axis spike.
		// Quiet room ~0.01, clap ~0.3-0.8, slap on desk ~0.5+
		// We amplify to match accelerometer-like magnitudes.
		amplitude := rms * 10.0

		samples <- Sample{
			X:    0,
			Y:    0,
			Z:    amplitude,
			Time: time.Now(),
		}
	}
}

func (s *MicSensor) Close() error {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	return nil
}
