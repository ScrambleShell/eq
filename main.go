package main

import (
	"fmt"
	"math/cmplx"
	"time"

	"github.com/mjibson/go-dsp/fft"
)

func convertTo64(ar []float32) []float64 {
	newar := make([]float64, len(ar))
	for i, v := range ar {
		newar[i] = float64(v)
	}
	return newar
}

// So far it looks like the returns from this FFT library are
// element 0: 0Hz
// element 1: 0.5 amp * happens once
// ...
// element n/2: 1 amp * happens n/2 times
// element n/2 + 1: 0.5 amp * happens n/2 - 1 times
// ...
// element n-1: 0.5 amp * happens once
// all are cosine funcs
// magnitude is * n
func fftMag(in []float32) []float64 {
	cmplxResults := fft.FFTReal(convertTo64(in))
	toReturn := make([]float64, (len(cmplxResults)/2)-1)
	for i := 1; i < len(cmplxResults)/2; i++ {
		toReturn[i-1] = cmplx.Abs(cmplxResults[i]) + cmplx.Abs(cmplxResults[len(cmplxResults)-i])
	}
	return toReturn
}

func binFreq(in []float64, bins int) []float64 {
	toReturn := make([]float64, bins)
	binSize := len(in) / bins
	for i, y := range in {
		bin := i / binSize
		if bin >= bins {
			continue
		}
		toReturn[bin] += y
	}
	return toReturn
}

func main() {
	audioInterface, err := NewPortAudioInterface()
	if err != nil {
		fmt.Println("Error creating audio interface", err)
		return
	}

	frameBuffer := make([]float32, 1602) // this should be 100ms worth
	for {
		time.Sleep(100 * time.Millisecond)
		n, err := audioInterface.ReadSamples(frameBuffer)
		if err != nil {
			fmt.Println("Read Error", err)
			return
		}

		if n < len(frameBuffer) {
			fmt.Println("Underread")
		}

		fmt.Println(n)
		go fmt.Println(binFreq(fftMag(frameBuffer), 10))

	}
}
