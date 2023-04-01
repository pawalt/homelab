package jobs

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v4"
	"github.com/pawalt/homelab/golinks/pkg/config"
)

func GetVitalActivity() {
	resp, err := http.Get("https://display.safespace.io/value/live/a7796f34")
	panicErr(err)
	if resp.StatusCode != 200 {
		panic(fmt.Sprintf("status code error: %d %s", resp.StatusCode, resp.Status))
	}

	b, err := ioutil.ReadAll(resp.Body)
	panicErr(err)

	count, err := strconv.Atoi(string(b))
	panicErr(err)

	q := `INSERT INTO timings (active_count) VALUES (%d)`
	q = fmt.Sprintf(q, count)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, config.CONNECTION_URL)
	panicErr(err)
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, q)
	panicErr(err)
}
