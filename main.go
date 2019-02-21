package main

import (
	"fmt"
	"math"
	"math/cmplx"
	"time"

	"github.com/mjibson/go-dsp/fft"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	width  = int32(640)
	height = int32(500)

	columns = 32
	//columns = 10
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

func binFreq(in []float64, bins int) []int {
	toReturn := make([]float64, bins)
	binSize := len(in) / bins
	for i, y := range in {
		bin := i / binSize
		if bin >= bins {
			continue
		}
		toReturn[bin] += y
	}

	toReturnInts := make([]int, bins)
	for i := range toReturn {
		toReturnInts[i] = int(math.Ceil(toReturn[i]))
	}
	return toReturnInts
}

func audioLoop(binChan chan []int) {
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

		binChan <- binFreq(fftMag(frameBuffer), columns)
	}
}

func main() {

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	colWidth := width / int32(columns)
	window, err := sdl.CreateWindow("EQ", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		width, height, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}
	surface.FillRect(nil, 0)

	rects := make([]sdl.Rect, columns)
	window.UpdateSurface()

	binChan := make(chan []int, 1)
	go audioLoop(binChan)

	running := true
	for running {
		select {
		case heights := <-binChan:
			surface.FillRect(nil, 0)
			for i := range rects {
				h := int32(heights[i])
				rects[i] = sdl.Rect{int32(i) * colWidth, height - h, colWidth, h}
			}
			surface.FillRects(rects, 0xffff0000)
			window.UpdateSurface()
		default:
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch event.(type) {
				case *sdl.QuitEvent:
					fmt.Println("Quit")
					running = false
					break
				}
			}
		}
	}
}
