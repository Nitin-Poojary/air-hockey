package utils

import (
	"os"
	"strconv"
)

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetEnvInt(key, defaultValue string) int {
	value := os.Getenv(key)
	if value == "" {
		val, _ := strconv.Atoi(defaultValue)
		return val
	}
	val, _ := strconv.Atoi(value)
	return val
}
