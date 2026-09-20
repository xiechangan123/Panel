package webserver

type Type string

const (
	TypeNginx         Type = "nginx"
	TypeApache        Type = "apache"
	TypeOpenLiteSpeed Type = "openlitespeed"
	TypeCaddy         Type = "caddy"
)
