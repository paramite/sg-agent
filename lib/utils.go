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

// GetTimestamp returns unix timestamp
func GetTimestamp() float64 {
	return float64(time.Now().Unix())
}

// GetHostname returns hostname if it is possible or "localhost" is not possilbe to obtain from OS.
func GetHostname() string {
	out := defaultHostname
	if host, err := os.Hostname(); err == nil {
		out = host
	}
	return out
}

// FormatPublisher returns standartized message publisher name
func FormatPublisher(suffix string) string {
	return strings.Join([]string{GetHostname(), suffix}, "-")
}

// FormatIndex returns standartized document index name
func FormatIndex(prefix string) string {
	year, month, day := time.Now().Date()
	return fmt.Sprintf("%s-%s.%d.%02d.%02d", prefix, strings.ReplaceAll(GetHostname(), "-", "_"), year, month, day)
}
