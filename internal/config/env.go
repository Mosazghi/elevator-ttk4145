package config

import "os"

var ProdMode = os.Getenv("ENV") == "production" || os.Getenv("ENV") == "prod"
