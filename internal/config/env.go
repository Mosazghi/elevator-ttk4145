package config

import "os"

var prodMode = os.Getenv("ENV") == "production" || os.Getenv("ENV") == "prod"

// IsProdMode reports whether the app is running in production mode.
func IsProdMode() bool {
	return prodMode
}
