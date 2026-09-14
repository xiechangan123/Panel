package conf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	cfg := &Config{}
	cfg.Add("listen", "80")
	cfg.Add("listen", "443", "ssl")
	server := cfg.AddBlock("server", "a")
	server.Add("root", "/var/www")
	server.AddBlock("location", "/").Add("try_files", "$uri")
	cfg.AddMeta("pass", "http://x")

	assert.Equal(t, "80", cfg.Value("listen"))
	assert.Len(t, cfg.GetAll("listen"), 2)
	assert.Equal(t, []string{"443", "ssl"}, cfg.GetAll("listen")[1].Values())
	assert.Equal(t, "/var/www", cfg.FindOne("server.root").Arg(0))
	assert.Equal(t, "$uri", cfg.FindOne("server.location.try_files").Arg(0))
	assert.Len(t, cfg.FindBlocks("server.location"), 1)
	assert.Equal(t, server, cfg.GetBlock("server", "a"))
	assert.Nil(t, cfg.GetBlock("server", "b"))
	assert.Equal(t, "http://x", cfg.Meta("pass"))

	// nil 安全的链式取值
	assert.Equal(t, "", cfg.Get("missing").Arg(0))
	assert.Nil(t, cfg.Get("listen").Get("x"))
	assert.Equal(t, 0, cfg.Get("listen").Len())

	cfg.Set("root", "/new")
	assert.Equal(t, "/new", cfg.Value("root"))
	assert.Equal(t, 2, cfg.Remove("listen"))
	assert.False(t, cfg.Has("listen"))
	assert.Equal(t, 1, cfg.RemoveFunc("server", func(d *Directive) bool { return d.Arg(0) == "a" }))
}

func TestSort(t *testing.T) {
	cfg := &Config{}
	cfg.Append(Cmt("c1"))
	cfg.Add("b")
	cfg.Add("a")
	cfg.Append(Cmt("tail"))
	order := map[string]int{"a": 0, "b": 1}
	sorted := Sort(cfg.Nodes, func(d *Directive) int { return order[d.Name] })
	assert.Equal(t, "a", sorted[0].(*Directive).Name)
	assert.Equal(t, " c1", sorted[1].(*Comment).Text)
	assert.Equal(t, "b", sorted[2].(*Directive).Name)
	assert.Equal(t, " tail", sorted[3].(*Comment).Text)
}
