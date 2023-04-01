package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v4"
	"github.com/pawalt/homelab/golinks/pkg/config"
)

// run this like `go run pkg/migrations/main.go 0_CREATE_DB`
func main() {
	hostsFunc := os.Args[1]

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, config.CONNECTION_URL)
	e(err)
	defer conn.Close(context.Background())

	migrationMap[hostsFunc](conn)

	fmt.Println("success")
}

var (
	migrationMap = map[string]func(*pgx.Conn){
		"0_CREATE_DB": func(conn *pgx.Conn) {
			panicExec(conn, `
CREATE TABLE redirects (
source STRING PRIMARY KEY,
target STRING
);
			`)
		},
		"1_CREATE_TIMINGS": func(conn *pgx.Conn) {
			panicExec(conn, `
CREATE TABLE timings (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
log_time TIMESTAMP DEFAULT NOW(),
active_count INT NOT NULL
);
			`)
		},
	}
)

func panicExec(conn *pgx.Conn, command string) {
	_, err := conn.Exec(context.Background(), command)
	if err != nil {
		panic(err)
	}
}

func e(err error) {
	if err != nil {
		panic(err)
	}
}
