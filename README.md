# fuck-linux

**Linux moans every time you hit it.**

A CLI tool that reacts to physical hits on a device and plays sounds.
Inspired by the macOS [spank](https://github.com/taigrr/spank) project — hardware-agnostic and extensible.

## 🚀 Quick Install

Run this one-liner to install the binary and default sounds:

```bash
curl -fsSL https://raw.githubusercontent.com/Sudhir878786/fuck-linux/Sudhir/install.sh | bash
```

## Features

- **Pluggable sensor backends:**
  - **IIO** — Linux Industrial I/O accelerometers (`/sys/bus/iio`)
  - **Serial** — Arduino over USB (newline-separated `X,Y,Z` values)
  - **Mic** — Microphone-based impact detection (via `arecord`)
- **Two playback modes:**
  - `random` — plays a random sound on each hit
  - `escalation` — sounds intensify with hit frequency
- **Low-latency** goroutine-based pipeline
- **Configurable** threshold, cooldown, speed, and volume scaling

## Project Structure

```
fuck-linux/
├── main.go              # CLI entry point (cobra)
├── sensor/
│   ├── sensor.go        # Sensor interface
│   ├── backends.go      # IIO, Serial, Mic implementations
│   └── util.go          # Shared helpers
├── detector/
│   ├── detector.go      # Spike detection + ring buffer
│   └── detector_test.go # Unit tests
├── audio/
│   └── player.go        # Playback, escalation tracker
├── tools/
│   └── generate_sounds.go  # Generate test WAV files
├── examples/
│   ├── arduino_accelerometer.ino  # Arduino sketch
│   └── sim_serial.go              # Serial simulator for testing
└── README.md
```

## Usage

### With microphone (recommended for most laptops)

```bash
./fuck-linux --source mic --sound-dir sounds/ --mode random --threshold 0.4
```

Now **hit or slap your laptop** — the mic detects the impact and plays a sound!

### With IIO accelerometer (e.g., laptop with built-in sensor)

```bash
sudo ./fuck-linux --source iio --sound-dir sounds/ --mode random
```

### With Arduino over serial

1. Flash [examples/arduino_accelerometer.ino](examples/arduino_accelerometer.ino) to your Arduino
2. Connect via USB (typically `/dev/ttyUSB0` or `/dev/ttyACM0`)
3. Run:

```bash
./fuck-linux --source serial --device /dev/ttyUSB0 --sound-dir sounds/ --mode escalation
```



## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `random` | Playback mode: `random` or `escalation` |
| `--threshold` | `0.05` | Minimum amplitude to trigger (0.0–1.0) |
| `--cooldown` | `750` | Cooldown between triggers (ms) |
| `--source` | `iio` | Sensor backend: `iio`, `serial`, `mic` |
| `--device` | auto | Device path/port (auto-detected if empty) |
| `--sound-dir` | required | Directory containing MP3/WAV audio files |
| `--speed` | `1.0` | Playback speed multiplier |
| `--volume-scaling` | `false` | Scale volume by impact amplitude |

## Arduino Input Format

The serial sensor expects newline-separated CSV values:

```
X,Y,Z
```

Where X, Y, Z are floating-point accelerometer values in m/s² (or any consistent unit).  
Lines starting with `#` are treated as comments and ignored.

Example output from Arduino at 100 Hz:
```
0.1200,-0.0300,9.8100
0.1500,-0.0100,9.7800
3.4200,1.2000,14.5000    ← spike = slap detected
0.1100,-0.0200,9.8050
```

## Detection Algorithm

1. Ring buffer (64 samples) tracks the moving average baseline
2. Each new sample's magnitude (`√(x²+y²+z²)`) is compared to the baseline
3. If the delta exceeds the threshold, and the cooldown has elapsed, an event fires
4. Events are classified by severity: `light` (<0.15), `medium` (0.15–0.5), `hard` (>0.5)

## Extending with New Sensors

Implement the `sensor.Sensor` interface:

```go
type Sensor interface {
    Start(samples chan<- Sample) error
    Close() error
    Name() string
}
```

Then add a case in `main.go`'s source switch to wire it up.

## License

MIT
