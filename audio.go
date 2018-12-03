package main

type AudioInterface interface {
	ReadSamples(p []float32) (int, error)
}
