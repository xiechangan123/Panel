package service

import "github.com/leonelquinteros/gotext"

type EnvironmentService struct {
	t *gotext.Locale
}

func NewEnvironmentService(t *gotext.Locale) *EnvironmentService {
	return &EnvironmentService{
		t: t,
	}
}
