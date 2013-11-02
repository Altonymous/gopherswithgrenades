package main

type benchmarkResponse struct {
	complete          int
	failed            int
	requestsPerSecond float32
	timePerRequest    float32
	err               []error
}
