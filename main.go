package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Sudhir878786/fuck-linux/audio"
	"github.com/Sudhir878786/fuck-linux/detector"
	"github.com/Sudhir878786/fuck-linux/sensor"
	"github.com/spf13/cobra"
)

var version = "dev"

var (
	flagMode      string
	flagThreshold float64
	flagCooldown  int
	flagSource    string
	flagDevice    string
	flagSoundDir  string
	flagSpeed     float64
	flagVolScale  bool
)

func main() {
	cmd := &cobra.Command{
		Use:   "fuck-linux",
		Short: "Reacts to physical hits on the device and plays sounds",
		Long: `fuck-linux detects physical impacts via accelerometer, serial, or microphone
input and plays audio responses.

Sensor sources:
  iio     - Linux IIO accelerometer (/sys/bus/iio)
  serial  - Arduino serial input (newline-separated X,Y,Z)
  mic     - Microphone-based impact detection (via arecord)

Modes:
  random     - Play a random sound on each hit
  escalation - Sounds intensify the more you hit within a time window`,
		Version:      version,
		RunE:         runCmd,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&flagMode, "mode", "random", "Playback mode: random or escalation")
	cmd.Flags().Float64Var(&flagThreshold, "threshold", 0.4, "Minimum amplitude to trigger (0.0-1.0)")
	cmd.Flags().IntVar(&flagCooldown, "cooldown", 750, "Cooldown between triggers in milliseconds")
	cmd.Flags().StringVar(&flagSource, "source", "mic", "Sensor source: iio, serial, or mic")
	cmd.Flags().StringVar(&flagDevice, "device", "", "Device path or port (auto-detected if empty)")
	cmd.Flags().StringVar(&flagSoundDir, "sound-dir", "", "Directory containing audio files (MP3/WAV)")
	cmd.Flags().Float64Var(&flagSpeed, "speed", 1.0, "Playback speed multiplier")
	cmd.Flags().BoolVar(&flagVolScale, "volume-scaling", false, "Scale volume by impact amplitude")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCmd(cmd *cobra.Command, args []string) error {
	// Validate mode
	var playMode audio.PlayMode
	switch strings.ToLower(flagMode) {
	case "random":
		playMode = audio.ModeRandom
	case "escalation":
		playMode = audio.ModeEscalation
	default:
		return fmt.Errorf("invalid --mode %q: must be 'random' or 'escalation'", flagMode)
	}

	// Validate threshold
	if flagThreshold < 0 || flagThreshold > 1 {
		return fmt.Errorf("--threshold must be between 0.0 and 1.0")
	}

	// Validate cooldown
	cooldown := time.Duration(flagCooldown) * time.Millisecond
	if cooldown <= 0 {
		return fmt.Errorf("--cooldown must be greater than 0")
	}

	// Resolve sound pack
	if flagSoundDir == "" {
		if _, err := os.Stat("sounds"); err == nil {
			flagSoundDir = "sounds"
		} else if _, err := os.Stat("/usr/share/fucklinux/sounds"); err == nil {
			flagSoundDir = "/usr/share/fucklinux/sounds"
		} else if snapDir := os.Getenv("SNAP"); snapDir != "" && func() bool { _, err := os.Stat(filepath.Join(snapDir, "sounds")); return err == nil }() {
			flagSoundDir = filepath.Join(snapDir, "sounds")
		} else {
			return fmt.Errorf("--sound-dir is required (tried 'sounds', '/usr/share/fucklinux/sounds', and '$SNAP/sounds')")
		}
	}

	absDir, err := filepath.Abs(flagSoundDir)
	if err != nil {
		return fmt.Errorf("resolving sound-dir: %w", err)
	}

	pack, err := audio.LoadDir("custom", absDir, playMode)
	if err != nil {
		return err
	}

	// Create sensor
	var s sensor.Sensor
	switch strings.ToLower(flagSource) {
	case "iio":
		s, err = sensor.NewIIOSensor(flagDevice)
	case "serial":
		s, err = sensor.NewSerialSensor(flagDevice)
	case "mic":
		s, err = sensor.NewMicSensor(flagDevice)
	default:
		return fmt.Errorf("invalid --source %q: must be 'iio', 'serial', or 'mic'", flagSource)
	}
	if err != nil {
		return fmt.Errorf("initializing %s sensor: %w", flagSource, err)
	}
	defer s.Close()

	// Create detector
	det := detector.New(detector.Config{
		MinAmplitude: flagThreshold,
		Cooldown:     cooldown,
	})

	// Create audio player
	player := audio.NewPlayer()
	player.VolumeScaling = flagVolScale
	player.SpeedRatio = flagSpeed

	// Create slap tracker
	tracker := audio.NewSlapTracker(pack, cooldown)

	// Setup signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("fuck-linux: listening via %s in %s mode (threshold=%.3f, cooldown=%dms)\n",
		s.Name(), flagMode, flagThreshold, flagCooldown)
	fmt.Println("Press Ctrl+C to quit.")

	// Start sensor reader goroutine
	samples := make(chan sensor.Sample, 256)
	sensorErr := make(chan error, 1)

	go func() {
		sensorErr <- s.Start(samples)
	}()

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nbye!")
			s.Close()
			return nil

		case err := <-sensorErr:
			if err != nil {
				return fmt.Errorf("sensor error: %w", err)
			}
			return nil

		case sample := <-samples:
			ev := det.Process(sample.X, sample.Y, sample.Z, sample.Time)
			if ev == nil {
				continue
			}

			now := time.Now()
			num, score := tracker.Record(now)
			file := tracker.SelectFile(score)

			fmt.Printf("slap #%d [%s amp=%.5f] -> %s\n",
				num, ev.Severity, ev.Amplitude, filepath.Base(file))

			go func(f string, a float64) {
				if err := player.Play(f, a); err != nil {
					fmt.Fprintf(os.Stderr, "fuck-linux: audio error: %v\n", err)
				}
			}(file, ev.Amplitude)
		}
	}
}
