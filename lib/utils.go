package lib

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultHostname = "localhost"
)

func GetTimestamp() float64 {
	return float64(time.Now().Unix())
}

func GetHostname() string {
	out := defaultHostname
	if host, err := os.Hostname(); err == nil {
		out = host
	}
	return out
}

func FormatPublisher(suffix string) string {
	return strings.Join([]string{GetHostname(), suffix}, "-")
}

func FormatIndex(prefix string) string {
	year, month, day := time.Now().Date()
	return fmt.Sprintf("%s-%s.%d.%02d.%02d", prefix, strings.ReplaceAll(GetHostname(), "-", "_"), year, month, day)
}
