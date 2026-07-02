package request

type CronCreate struct {
	Name     string            `form:"name" json:"name" validate:"required && not_exists:crons,name"`
	Type     string            `form:"type" json:"type" validate:"required && in:shell,backup,cutoff,url,synctime"`
	Time     string            `form:"time" json:"time" validate:"required && cron"`
	Script   string            `form:"script" json:"script"`
	SubType  string            `form:"sub_type" json:"sub_type" validate:"required_if:Type,backup,cutoff"`
	Flock    bool              `form:"flock" json:"flock"`
	Storage  uint              `form:"storage" json:"storage"`
	Targets  []string          `form:"targets" json:"targets" validate:"required_if:Type,backup,cutoff"`
	Keep     uint              `form:"keep" json:"keep" validate:"required"`
	URL      string            `form:"url" json:"url"`
	Method   string            `form:"method" json:"method"`
	Headers  map[string]string `form:"headers" json:"headers"`
	Body     string            `form:"body" json:"body"`
	Timeout  uint              `form:"timeout" json:"timeout"`
	Insecure bool              `form:"insecure" json:"insecure"`
	Retries  uint              `form:"retries" json:"retries"`
}

type CronUpdate struct {
	ID       uint              `form:"id" json:"id" validate:"required && exists:crons,id"`
	Name     string            `form:"name" json:"name" validate:"required"`
	Type     string            `form:"type" json:"type" validate:"required && in:shell,backup,cutoff,url,synctime"`
	Time     string            `form:"time" json:"time" validate:"required && cron"`
	Script   string            `form:"script" json:"script"`
	SubType  string            `form:"sub_type" json:"sub_type"`
	Flock    bool              `form:"flock" json:"flock"`
	Storage  uint              `form:"storage" json:"storage"`
	Targets  []string          `form:"targets" json:"targets"`
	Keep     uint              `form:"keep" json:"keep"`
	URL      string            `form:"url" json:"url"`
	Method   string            `form:"method" json:"method"`
	Headers  map[string]string `form:"headers" json:"headers"`
	Body     string            `form:"body" json:"body"`
	Timeout  uint              `form:"timeout" json:"timeout"`
	Insecure bool              `form:"insecure" json:"insecure"`
	Retries  uint              `form:"retries" json:"retries"`
}

type CronStatus struct {
	ID     uint `form:"id" json:"id" validate:"required && exists:crons,id"`
	Status bool `form:"status" json:"status"`
}
