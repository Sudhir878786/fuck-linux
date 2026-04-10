# fuck-linux

**Linux moans every time you hit it.**

## 📺 See it in Action
*Experience the technical despair in real-time.*

<div align="center">
  <video src="demo.mp4" width="100%" controls></video>
</div>


A CLI tool that reacts to physical hits on your device (via microphone or accelerometer) and plays sounds. 

## 🚀 Installation

### Using Snap (Recommended)
You can seamlessly install `fuck-linux` via the Ubuntu Snap Store:
```bash
sudo snap install fuck-linux
```

### Using Install Script
Alternatively, run this one-liner to install the binary directly:
```bash
curl -fsSL https://raw.githubusercontent.com/Sudhir878786/fuck-linux/Sudhir/install.sh | bash
```

## 💻 Usage

Simply start the program from your terminal to use your device's microphone as a sensor:
```bash
fuck-linux
```
Now **hit or slap your laptop** — the mic detects the impact and plays a sound!

### Sensitivity & Threshold
You can adjust how hard you need to hit your device using the `--threshold` flag (between `0.0` and `1.0`):
- **Very sensitive (light taps):** `fuck-linux --threshold 0.1`
- **Default:** `fuck-linux --threshold 0.4`
- **Requires a hard hit:** `fuck-linux --threshold 0.8`

### All CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `random` | Playback mode: `random` or `escalation` |
| `--threshold` | `0.40` | Minimum amplitude to trigger (0.0–1.0) |
| `--cooldown` | `750` | Cooldown between triggers (ms) |
| `--source` | `mic` | Sensor backend: `iio`, `mic` |
| `--speed` | `1.0` | Playback speed multiplier |
| `--volume-scaling` | `false` | Scale volume by impact amplitude |
