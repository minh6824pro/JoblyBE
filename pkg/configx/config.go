package configx

import (
	"github.com/spf13/cast"
	"google.golang.org/protobuf/types/known/durationpb"
	"os"
	"strings"
	"time"
)

func GetEnvOrString(key string, defaultVal string) string {
	if os.Getenv(key) != "" {
		return os.Getenv(key)
	}
	return defaultVal
}

func GetEnvOrStrings(key string, defaultValue []string) []string {
	if os.Getenv(key) != "" {
		if s := os.Getenv(key); s != "" {
			return strings.Split(s, ",")
		}
	}
	return defaultValue
}

func GetEnvOrInt64(key string, defaultValue int64) int64 {
	if os.Getenv(key) != "" {
		return cast.ToInt64(os.Getenv(key))
	}
	return defaultValue
}

func GetEnvOrBool(key string, defaultValue bool) bool {
	if os.Getenv(key) != "" {
		return cast.ToBool(os.Getenv(key))
	}
	return defaultValue
}

func GetEnvOrDuration(key string, defaultValue *durationpb.Duration) time.Duration {
	if os.Getenv(key) != "" {
		d, err := time.ParseDuration(os.Getenv(key))
		if err == nil {
			return d
		}
	}
	if defaultValue != nil {
		return defaultValue.AsDuration()
	}
	return 0
}

func GetEnvOrInt(key string, defaultValue int) int {
	if os.Getenv(key) != "" {
		return cast.ToInt(os.Getenv(key))
	}
	return defaultValue
}
