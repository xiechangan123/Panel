package request

import "encoding/json"

type NotifyChannelCreate struct {
	Name    string          `json:"name" form:"name" validate:"required"`
	Type    string          `json:"type" form:"type" validate:"required && in:smtp"`
	Config  json.RawMessage `json:"config" form:"config"`
	Enabled bool            `json:"enabled" form:"enabled"`
}

type NotifyChannelUpdate struct {
	ID      uint            `json:"id" form:"id" uri:"id" validate:"required && exists:notify_channels,id"`
	Name    string          `json:"name" form:"name" validate:"required"`
	Type    string          `json:"type" form:"type" validate:"required && in:smtp"`
	Config  json.RawMessage `json:"config" form:"config"`
	Enabled bool            `json:"enabled" form:"enabled"`
}

type NotifySetting struct {
	Events   []string `json:"events" form:"events"`
	Channels []uint   `json:"channels" form:"channels"`
}
