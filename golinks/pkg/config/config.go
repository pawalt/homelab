package config

import (
	"fmt"
	"os"
)

var (
	CONNECTION_URL = fmt.Sprintf(
		"postgresql://peyton:%s@free-tier11.gcp-us-east1.cockroachlabs.cloud:26257/golinks?sslmode=verify-full&options=--cluster%%3Dtail-scale-2664",
		passwd(),
	)
)

func passwd() string {
	shaboom, exists := os.LookupEnv("DB_PASSWD")
	if !exists {
		panic("please provide DB_PASSWD")
	}
	return shaboom
}
