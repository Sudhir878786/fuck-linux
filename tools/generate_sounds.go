// generate_sounds is a helper tool that creates simple WAV beep files
// for testing spank without real audio assets.
//
// Usage: go run tools/generate_sounds.go <output_dir> <count>
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <output_dir> <count>\n", os.Args[0])
		os.Exit(1)
	}
	dir := os.Args[1]
	count, err := strconv.Atoi(os.Args[2])
	if err != nil || count < 1 {
		fmt.Fprintf(os.Stderr, "count must be a positive integer\n")
		os.Exit(1)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	for i := 0; i < count; i++ {
		// Each file is a short beep at an increasing frequency
		freq := 220.0 + float64(i)*80.0 // A3 + offset
		duration := 0.3 + float64(i)*0.05
		name := fmt.Sprintf("%s/hit_%02d.wav", dir, i+1)

		if err := writeWAV(name, freq, duration); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("generated %s (%.0f Hz, %.2fs)\n", name, freq, duration)
	}
}

func writeWAV(path string, freq, duration float64) error {
	sampleRate := 44100
	numSamples := int(duration * float64(sampleRate))
	dataSize := numSamples * 2 // 16-bit mono

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// RIFF header
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte("WAVE"))

	// fmt chunk
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))        // chunk size
	binary.Write(f, binary.LittleEndian, uint16(1))          // PCM
	binary.Write(f, binary.LittleEndian, uint16(1))          // mono
	binary.Write(f, binary.LittleEndian, uint32(sampleRate)) // sample rate
	binary.Write(f, binary.LittleEndian, uint32(sampleRate*2)) // byte rate
	binary.Write(f, binary.LittleEndian, uint16(2))          // block align
	binary.Write(f, binary.LittleEndian, uint16(16))         // bits per sample

	// data chunk
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	// Generate sine wave with fade in/out
	fadeLen := int(0.01 * float64(sampleRate)) // 10ms fade
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		sample := math.Sin(2.0 * math.Pi * freq * t)

		// Apply envelope
		envelope := 1.0
		if i < fadeLen {
			envelope = float64(i) / float64(fadeLen)
		} else if i > numSamples-fadeLen {
			envelope = float64(numSamples-i) / float64(fadeLen)
		}
		sample *= envelope * 0.8

		// Convert to int16
		val := int16(sample * 32767)
		binary.Write(f, binary.LittleEndian, val)
	}

	return nil
}
