package main

import (
	"fmt"
	"time"
)

func main() {
	audioInterface, err := NewPortAudioInterface()
	if err != nil {
		fmt.Println("Error creating audio interface", err)
		return
	}

	frameBuffer := make([]float32, 1600) // this should be 100ms worth
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

	}
}
