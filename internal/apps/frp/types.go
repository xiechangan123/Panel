package frp

import "regexp"

var (
	userCaptureRegex  = regexp.MustCompile(`(?m)^User=(.*)$`)
	groupCaptureRegex = regexp.MustCompile(`(?m)^Group=(.*)$`)
	userRegex         = regexp.MustCompile(`(?m)^User=.*$`)
	groupRegex        = regexp.MustCompile(`(?m)^Group=.*$`)
	serviceRegex      = regexp.MustCompile(`(?m)^\[Service\]$`)
)

type UserInfo struct {
	User  string `json:"user"`
	Group string `json:"group"`
}
