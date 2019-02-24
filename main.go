package main

import (
	"fmt"
	"math"
	"math/cmplx"
	"time"

	"github.com/bhmorse/eq/portaudio"
	"github.com/mjibson/go-dsp/fft"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	width  = int32(930)
	height = int32(800)

	// THIS IS NOW HARDCODED IN BINNING CODE
	columns = 31
	//columns = 10
)

func convertTo64(ar []int32) []float64 {
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
func fftMag(in []int32) []float64 {
	cmplxResults := fft.FFTReal(convertTo64(in))
	toReturn := make([]float64, (len(cmplxResults)/2)-1)
	for i := 1; i < len(cmplxResults)/2; i++ {
		toReturn[i-1] = cmplx.Abs(cmplxResults[i]) + cmplx.Abs(cmplxResults[len(cmplxResults)-i])
	}
	return toReturn
}

// https://www.presonus.com/learn/technical-articles/What-Is-a-Graphic-Eq
/* 31 bin:
18k   - 22.05k  ~4.1k
14.5k - 18k     ~3.5k
11.5k - 14.5k   ~3k
9k    - 11.5k   ~2.5k
7.2k  - 9k      ~1.8k
5.7k  - 7.2k    ~1.5k
4.5k  - 5.7k    ~1.2k
3.6k  - 4.5k    ~900
2.85k - 3.6k    ~750
2.25k - 2.85k   ~600
1.8k  - 2.25k   ~450
1.45k - 1.8k    ~350
1.15k - 1.45k   ~300
900   - 1.15k   ~250
720   - 900     ~180
570   - 720     ~150
450   - 570     ~120
360   - 450     ~90
285   - 360     ~75
225   - 285     ~60
180   - 225     ~45
142.5 - 180     ~37.5
112.5 - 142.5   ~30
90    - 112.5   ~22.5
71.5  - 90      ~18.5
56.5  - 71.5    ~15
45    - 56.5    ~11.5
36    - 45      ~9
28.5  - 36      ~7.5
22.5  - 28.5    ~6
17.5  - 22.5    ~5
*/
func binFreq(in []float64) []int {
	binBottoms := []float64{
		18000.0,
		14500.0,
		11500.0,
		9000.0,
		7200.0,
		5700.0,
		4500.0,
		3600.0,
		2850.0,
		2250.0,
		1800.0,
		1450.0,
		1150.0,
		900.0,
		720.0,
		570.0,
		450.0,
		360.0,
		285.0,
		225.0,
		180.0,
		142.5,
		112.5,
		90.0,
		71.5,
		56.5,
		45.0,
		36.0,
		28.5,
		22.5,
		17.5,
	}
	toReturn := make([]float64, 31)
	for i, y := range in {
		for j, bottom := range binBottoms {
			if float64(i*5.0) > bottom {
				toReturn[30-j] += y / 8000000000.0
				break
			}
		}
	}

	fmt.Println(toReturn)
	toReturnInts := make([]int, 31)
	for i := range toReturn {
		toReturnInts[i] = int(math.Ceil(toReturn[i]))
	}
	return toReturnInts
}

func audioLoop(binChan chan []int) {
	audioInterface, err := portaudio.NewPortAudioInterface()
	if err != nil {
		fmt.Println("Error creating audio interface", err)
		return
	}

	frameBuffer := make([]int32, 8820*2) // this should be 200 ms worth
	readBuffer := make([]int32, 8820*2)
	for {
		time.Sleep(100 * time.Millisecond)
		n, err := audioInterface.ReadSamples(readBuffer)
		if err != nil {
			fmt.Println("Read Error", err)
			continue
		}
		frameBuffer = frameBuffer[n:]
		frameBuffer = append(frameBuffer, readBuffer[:n]...)

		binChan <- binFreq(fftMag(frameBuffer))
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
