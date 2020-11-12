package env

import "os"

var Env = os.Getenv("ENV")

func IsProd() bool {
	return Env != "local"
}

func GetEnv() string {
	return Env
}
