# Testing Guide — spank (Linux)

This guide covers how to build, test, and verify the spank project on a Linux system.

---

## 1. Prerequisites

### System packages

```bash
sudo apt update
sudo apt install -y alsa-utils libasound2-dev golang
```

### Verify Go version (1.21+ required)

```bash
go version
```

---

## 2. Build

```bash
cd spank
go mod tidy
go build -o spank .
```

You should see a `spank` binary in the current directory:

```bash
ls -lh spank
```

---

## 3. Generate Test Sounds

The project does not ship audio files. Generate WAV test tones:

```bash
go run tools/generate_sounds.go sounds/ 10
```

This creates 10 WAV files (`hit_01.wav` through `hit_10.wav`) at increasing frequencies in the `sounds/` directory.

Verify:

```bash
ls sounds/
# Expected: hit_01.wav  hit_02.wav  ... hit_10.wav
```

---

## 4. Run Unit Tests

```bash
go test ./...
```

This runs the detector tests which verify:

- `TestMagnitude` — Euclidean norm calculation
- `TestClassifySeverity` — amplitude-to-severity mapping
- `TestDetectorBasicSpike` — spike detection against a stable baseline
- `TestDetectorCooldown` — events are suppressed during cooldown window
- `TestDrainEvents` — event buffer drain/reset

Expected output:

```
ok   github.com/spank-linux/spank/detector  0.XXXs
```

---

## 5. Test Without Hardware (Simulated Serial)

This is the easiest way to verify the full pipeline works — no accelerometer or microphone needed.

### Terminal 1 — Run spank with simulated serial input

```bash
go run examples/sim_serial.go | ./spank --source serial --device /dev/stdin --sound-dir sounds/ --mode random
```

### What to expect

- The simulator generates baseline accelerometer readings at 100 Hz
- Every ~3 seconds, it injects a random spike (simulated slap)
- You should see output like:

```
spank: listening via serial:/dev/stdin in random mode (threshold=0.050, cooldown=750ms)
Press Ctrl+C to quit.
slap #1 [medium amp=0.23450] -> hit_07.wav
slap #2 [hard amp=0.61200] -> hit_03.wav
slap #3 [light amp=0.08900] -> hit_05.wav
```

- Audio plays through your default ALSA output device

Press `Ctrl+C` to stop.

---

## 6. Test With Microphone (Physical Hit Detection)

If your Linux machine has a built-in or USB microphone:

```bash
# Verify mic works
arecord -d 2 -f S16_LE -r 44100 -c 1 /tmp/test.wav && aplay /tmp/test.wav

# Run spank in mic mode
./spank --source mic --sound-dir sounds/ --mode random --threshold 0.3
```

Now **physically hit or slap the surface** near the microphone. Each impact should trigger a sound.

### Tuning the threshold

| Threshold | Sensitivity |
|-----------|-------------|
| `0.1`     | Very sensitive — may trigger on ambient noise |
| `0.3`     | Good default — detects firm taps and slaps |
| `0.5`     | Hard hits only |
| `0.8`     | Very hard impacts only |

Adjust with `--threshold`:

```bash
./spank --source mic --sound-dir sounds/ --threshold 0.2
```

---

## 7. Test With IIO Accelerometer

Check if your laptop has an IIO accelerometer:

```bash
ls /sys/bus/iio/devices/*/in_accel_x_raw 2>/dev/null
```

If a path is printed, you have one. Run:

```bash
sudo ./spank --source iio --sound-dir sounds/ --mode escalation
```

(`sudo` is often required for IIO device access.)

Hit or shake the laptop — sounds escalate with hit frequency.

---

## 8. Test With Arduino (Serial)

1. Flash [examples/arduino_accelerometer.ino](examples/arduino_accelerometer.ino) to an Arduino with an accelerometer (LIS3DH, MPU6050, ADXL345)
2. Connect via USB
3. Find the serial port:

```bash
ls /dev/ttyUSB* /dev/ttyACM* 2>/dev/null
```

4. Run:

```bash
./spank --source serial --device /dev/ttyUSB0 --sound-dir sounds/ --mode escalation
```

5. Tap or hit the accelerometer — sounds play on impact.

---

## 9. Test Escalation Mode

Escalation mode increases sound intensity when you hit rapidly:

```bash
go run examples/sim_serial.go | ./spank --source serial --device /dev/stdin --sound-dir sounds/ --mode escalation
```

In escalation mode:
- `hit_01.wav` plays for occasional hits (low score)
- `hit_10.wav` plays during rapid sustained hitting (high score)
- Intensity decays over ~30 seconds of inactivity

---

## 10. Test Volume Scaling

```bash
go run examples/sim_serial.go | ./spank --source serial --device /dev/stdin --sound-dir sounds/ --volume-scaling
```

Harder hits play louder, softer hits play quieter.

---

## 11. Test Speed Control

```bash
# 1.5x playback speed
go run examples/sim_serial.go | ./spank --source serial --device /dev/stdin --sound-dir sounds/ --speed 1.5

# Half speed
go run examples/sim_serial.go | ./spank --source serial --device /dev/stdin --sound-dir sounds/ --speed 0.5
```

---

## 12. All CLI Flags Reference

```bash
./spank --help
```

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `random` | `random` or `escalation` |
| `--threshold` | `0.05` | Trigger sensitivity (0.0–1.0) |
| `--cooldown` | `750` | Min ms between triggers |
| `--source` | `iio` | `iio`, `serial`, or `mic` |
| `--device` | auto | Device path or ALSA device |
| `--sound-dir` | *required* | Directory of MP3/WAV files |
| `--speed` | `1.0` | Playback speed multiplier |
| `--volume-scaling` | `false` | Scale volume by hit strength |

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `speaker init` error | Install `libasound2-dev` and rebuild |
| `arecord: command not found` | `sudo apt install alsa-utils` |
| No sound output | Check `aplay -l` for audio devices; try `pulseaudio --start` |
| IIO permission denied | Run with `sudo` or add user to `iio` group |
| Serial permission denied | `sudo chmod 666 /dev/ttyUSB0` or add user to `dialout` group |
| Mic too sensitive | Increase `--threshold` (e.g., `0.5`) |
| No slaps detected | Decrease `--threshold` (e.g., `0.02`) |
| Sounds overlap/cut off | Increase `--cooldown` (e.g., `1000`) |
