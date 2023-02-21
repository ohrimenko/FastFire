package config

import (
	"os"
	"path/filepath"
	"time"
)

var (
	// Time allowed to write a message to the peer.
	WriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	PongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	PingPeriod = int64((PongWait * 9) / 10)

	// Maximum message size allowed from peer.
	MaxMessageSize = int64(512)
)

type Config struct{}

var App Config

func Env(key string) string {
	return os.Getenv(key)
}

// GetEnv retrieves the environment variable key. If it does not exist it returns the default.
func GetEnv(key string, dfault string, combineWith ...string) string {
	value := os.Getenv(key)
	if value == "" {
		value = dfault
	}

	switch len(combineWith) {
	case 0:
		return value
	case 1:
		return filepath.Join(value, combineWith[0])
	default:
		all := make([]string, len(combineWith)+1)
		all[0] = value
		copy(all[1:], combineWith)
		return filepath.Join(all...)
	}
}
