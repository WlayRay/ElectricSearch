package util

import (
	"context"
	"time"
)

func GetDefaultTimeoutContext() (context.Context, context.CancelFunc) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	return timeoutCtx, cancel
}
