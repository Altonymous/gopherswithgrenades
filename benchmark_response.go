package main

// -Complete requests:      5
// -Failed requests:        4
// -Requests per second:    5.19 [#/sec] (mean)
// -Time per request:       192.649 [ms] (mean)
type benchmarkResponse struct {
	Complete          int
	Failed            int
	RequestsPerSecond float32
	TimePerRequest    float32
	err               []error
}
