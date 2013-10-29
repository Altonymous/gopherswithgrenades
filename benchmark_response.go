package main

type benchmarkResponse struct {
	Complete          int
	Failed            int
	RequestsPerSecond float32
	TimePerRequest    float32
	err               []error
}
