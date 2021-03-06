package portaudio

import (
	"errors"
	"fmt"

	"github.com/gordonklaus/portaudio"
)

const (
	InputChannels  = 2
	OutputChannels = 2
	SampleRate     = 44100
)

type PortAudioInterface struct {
	stream      *portaudio.Stream
	frameBuffer []int32
}

func NewPortAudioInterface() (*PortAudioInterface, error) {
	portaudio.Initialize()
	h, err := portaudio.DefaultHostApi()
	if err != nil {
		fmt.Println("Error establishing portaudio host", err)
		return nil, err
	}

	p := portaudio.LowLatencyParameters(h.DefaultInputDevice, h.DefaultOutputDevice)
	p.Input.Channels = InputChannels
	p.Output.Channels = OutputChannels
	p.SampleRate = SampleRate
	writer := &PortAudioInterface{
		frameBuffer: make([]int32, 0),
	}
	writer.stream, err = portaudio.OpenStream(p, writer.processAudio)
	if err != nil {
		fmt.Println("Error opening portaudio stream", err)
		return nil, err
	}
	err = writer.stream.Start()
	if err != nil {
		fmt.Println("Error starting portaudio stream", err)
		return nil, err
	}
	return writer, nil
}

func (p *PortAudioInterface) processAudio(in, out []int32) {
	//copy(out, in)
	p.frameBuffer = append(p.frameBuffer, in...)
}

func (p *PortAudioInterface) ReadSamples(buf []int32) (n int, err error) {
	if len(p.frameBuffer) >= len(buf) {
		copy(buf, p.frameBuffer[:len(buf)])
		p.frameBuffer = p.frameBuffer[len(buf):]
		return len(buf), nil
	} else {
		copy(buf, p.frameBuffer)
		n := len(p.frameBuffer)
		p.frameBuffer = make([]int32, 0)
		return n, nil
	}
	return 0, errors.New("Unknown read error")
}
