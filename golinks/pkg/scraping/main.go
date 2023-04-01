package main

import "github.com/pawalt/homelab/golinks/pkg/jobs"

func main() {
	jobs.GetVitalActivity()
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
