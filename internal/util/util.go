package util

import (
	"os"
	"strconv"

	"github.com/airbrake/gobrake"
)

// Airbrake logging
var Airbrake *gobrake.Notifier

func init() {
	projectID, err := strconv.ParseInt(GetEnvOr("AIRBRAKE_PROJECT_ID", "123456"), 10, 64)

	if err != nil {
		panic(err)
	}
	Airbrake = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   projectID,
		ProjectKey:  GetEnvOr("AIRBRAKE_API_KEY", "FIXME"),
		Environment: GetEnvOr("AIRBRAKE_ENVIRONMENT", "production"),
	})
}

// GetEnvOr returns key or default value
func GetEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
