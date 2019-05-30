package controller

import (
	"time"
)

const ANNOTATION_TIMEOUT = "controller-zero-scaler/idleTimeout"
const ANNOTATION_WATCHED_KINDS = "controller-zero-scaler/watchedKinds"
const ANNOTATION_OWNED_KINDS = "controller-zero-scaler/ownedKinds"

func parseTimeout() {
	time.ParseDuration("10h")
}
