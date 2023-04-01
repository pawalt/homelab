package jobs

import (
	"github.com/robfig/cron/v3"
)

func GetJobs() (*cron.Cron, error) {
	c := cron.New(cron.WithSeconds())

	// Record a count of active vital ppl every 15 minutes
	_, err := c.AddFunc("0 */15 * * * *", GetVitalActivity)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
