// sim_serial is a tool that simulates Arduino serial output for testing.
// It writes newline-separated X,Y,Z accelerometer values to stdout,
// with periodic synthetic "slap" spikes.
//
// Usage: go run examples/sim_serial.go | ./fuck-linux --source serial --device /dev/stdin --sound-dir sounds/
package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	ticker := time.NewTicker(10 * time.Millisecond) // 100 Hz
	defer ticker.Stop()

	slapInterval := 3 * time.Second
	lastSlap := time.Now()

	for range ticker.C {
		// Baseline: gravity on Z axis with mild noise
		x := (rand.Float64() - 0.5) * 0.02
		y := (rand.Float64() - 0.5) * 0.02
		z := 9.81 + (rand.Float64()-0.5)*0.05

		// Inject a slap spike at random intervals
		if time.Since(lastSlap) > slapInterval {
			if rand.Float64() < 0.1 { // ~10% chance per tick once interval passed
				spike := 2.0 + rand.Float64()*5.0
				axis := rand.Intn(3)
				switch axis {
				case 0:
					x += spike * (2*float64(rand.Intn(2)) - 1)
				case 1:
					y += spike * (2*float64(rand.Intn(2)) - 1)
				case 2:
					z += spike
				}
				lastSlap = time.Now()
			}
		}

		fmt.Printf("%.4f,%.4f,%.4f\n", x, y, z)
	}
}
