package audio

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

// PlayMode determines how the next sound file is selected.
type PlayMode int

const (
	// ModeRandom selects a random file from the pack.
	ModeRandom PlayMode = iota
	// ModeEscalation selects files based on intensity score.
	ModeEscalation
)

// SoundPack holds a set of audio files for playback.
type SoundPack struct {
	Name  string
	Mode  PlayMode
	Files []string // sorted file paths
}

// audioExts lists supported audio file extensions.
var audioExts = map[string]bool{
	".mp3": true, ".wav": true, ".ogg": true, ".flac": true,
}

// LoadDir loads audio files (mp3, wav, ogg, flac) from a directory.
func LoadDir(name, dir string, mode PlayMode) (*SoundPack, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading sound dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !audioExts[ext] {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)

	if len(files) == 0 {
		return nil, fmt.Errorf("no audio files (.mp3/.wav/.ogg/.flac) found in %s", dir)
	}

	return &SoundPack{Name: name, Mode: mode, Files: files}, nil
}

// SlapTracker tracks slap intensity over time for escalation mode.
type SlapTracker struct {
	mu       sync.Mutex
	score    float64
	lastTime time.Time
	total    int
	halfLife float64
	scale    float64
	pack     *SoundPack
}

const decayHalfLife = 30.0

// NewSlapTracker creates a tracker for the given sound pack and cooldown.
func NewSlapTracker(pack *SoundPack, cooldown time.Duration) *SlapTracker {
	cooldownSec := cooldown.Seconds()
	ssMax := 1.0 / (1.0 - math.Pow(0.5, cooldownSec/decayHalfLife))
	scale := (ssMax - 1) / math.Log(float64(len(pack.Files)+1))
	return &SlapTracker{
		halfLife: decayHalfLife,
		scale:    scale,
		pack:     pack,
	}
}

// Record a new slap. Returns the total count and current intensity score.
func (st *SlapTracker) Record(now time.Time) (int, float64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if !st.lastTime.IsZero() {
		elapsed := now.Sub(st.lastTime).Seconds()
		st.score *= math.Pow(0.5, elapsed/st.halfLife)
	}
	st.score += 1.0
	st.lastTime = now
	st.total++
	return st.total, st.score
}

// SelectFile picks the appropriate audio file based on mode and score.
func (st *SlapTracker) SelectFile(score float64) string {
	if st.pack.Mode == ModeRandom {
		return st.pack.Files[rand.Intn(len(st.pack.Files))]
	}

	// Escalation mode: exponential mapping
	maxIdx := len(st.pack.Files) - 1
	idx := int(float64(len(st.pack.Files)) * (1.0 - math.Exp(-(score-1)/st.scale)))
	if idx > maxIdx {
		idx = maxIdx
	}
	if idx < 0 {
		idx = 0
	}
	return st.pack.Files[idx]
}

var (
	speakerMu   sync.Mutex
	speakerInit bool
	speakerRate beep.SampleRate
)

// Player handles audio playback.
type Player struct {
	VolumeScaling bool
	SpeedRatio    float64
}

// NewPlayer creates a new audio player with default settings.
func NewPlayer() *Player {
	return &Player{SpeedRatio: 1.0}
}

// Play decodes and plays the given audio file. Blocks until playback completes.
func (p *Player) Play(path string, amplitude float64) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	return p.playReader(f, path, amplitude)
}

// PlayEmbedded plays audio from embedded bytes.
func (p *Player) PlayEmbedded(data []byte, name string, amplitude float64) error {
	return p.playReader(io.NopCloser(bytes.NewReader(data)), name, amplitude)
}

func (p *Player) playReader(r io.ReadCloser, name string, amplitude float64) error {
	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error

	// Try MP3 first, then WAV
	streamer, format, err = mp3.Decode(r)
	if err != nil {
		// Reset if possible and try WAV
		if seeker, ok := r.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
			streamer, format, err = wav.Decode(r)
		}
		if err != nil {
			return fmt.Errorf("decode %s: %w", name, err)
		}
	}
	defer streamer.Close()

	speakerMu.Lock()
	if !speakerInit {
		if initErr := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); initErr != nil {
			speakerMu.Unlock()
			return fmt.Errorf("speaker init: %w", initErr)
		}
		speakerRate = format.SampleRate
		speakerInit = true
	}
	speakerMu.Unlock()

	var source beep.Streamer = streamer

	// Resample if file's sample rate doesn't match the speaker's
	if format.SampleRate != speakerRate {
		source = beep.Resample(4, format.SampleRate, speakerRate, source)
	}

	if p.VolumeScaling {
		source = &effects.Volume{
			Streamer: source,
			Base:     2,
			Volume:   amplitudeToVolume(amplitude),
			Silent:   false,
		}
	}

	if p.SpeedRatio != 1.0 && p.SpeedRatio > 0 {
		fakeRate := beep.SampleRate(int(float64(speakerRate) * p.SpeedRatio))
		source = beep.Resample(4, fakeRate, speakerRate, source)
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(source, beep.Callback(func() {
		done <- true
	})))
	<-done
	return nil
}

// amplitudeToVolume maps detected amplitude to a beep volume level.
func amplitudeToVolume(amplitude float64) float64 {
	const (
		minAmp = 0.05
		maxAmp = 0.80
		minVol = -3.0
		maxVol = 0.0
	)

	if amplitude <= minAmp {
		return minVol
	}
	if amplitude >= maxAmp {
		return maxVol
	}

	t := (amplitude - minAmp) / (maxAmp - minAmp)
	t = math.Log(1+t*99) / math.Log(100)
	return minVol + t*(maxVol-minVol)
}
