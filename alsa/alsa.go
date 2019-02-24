package alsa

import (
	"fmt"

	"github.com/cocoonlife/goalsa"
)

const (
	periodFrames     = 2048
	periods          = 4
	numIntSamples    = periodFrames * 2
	numBufferSamples = 4
)

func bufferParams() alsa.BufferParams {
	return alsa.BufferParams{
		PeriodFrames: periodFrames,
		Periods:      periods,
	}
}

type ALSAInterface struct {
	capture  *alsa.CaptureDevice
	playback *alsa.PlaybackDevice
}

func NewALSAInterface() (*ALSAInterface, error) {
	toReturn := ALSAInterface{}
	bufferParams := bufferParams()
	var err error
	toReturn.playback, err = alsa.NewPlaybackDevice("hw:0", 1, alsa.FormatS32LE, 44100, bufferParams)
	if err != nil {
		fmt.Println("Error establishing ALSA playback device", err)
		return nil, err
	}
	toReturn.capture, err = alsa.NewCaptureDevice("hw:0", 1, alsa.FormatS32LE, 44100, bufferParams)
	if err != nil {
		fmt.Println("Error establishing ALSA capture device", err)
		return nil, err
	}
	return &toReturn, nil
}

func (a *ALSAInterface) ReadSamples(buf []int32) (n int, err error) {
	return a.capture.Read(buf)
}
