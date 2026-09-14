package caddy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// 主机名与 URL 重定向各写一个匹配器加 redir，404 重定向写在 handle_errors 块里

func redirectStatus(r types.Redirect) int {
	if r.StatusCode == 0 {
		return 308
	}
	return r.StatusCode
}

func (v *baseVhost) buildRedirects(body *conf.Block) {
	for i, r := range v.redirects {
		name := fmt.Sprintf("@ace_redirect_%d", i)
		to := r.To
		if r.KeepURI {
			to += "{uri}"
		}
		code := strconv.Itoa(redirectStatus(r))
		switch r.Type {
		case types.RedirectTypeHost:
			body.Add(name, "host", r.From)
			body.Add("redir", name, to, code)
		case types.RedirectTypeURL:
			body.Add(name, "path", r.From)
			body.Add("redir", name, to, code)
		case types.RedirectType404:
			body.AddBlock("handle_errors", "404").Add("redir", to, code)
		}
	}
}

func (v *baseVhost) loadRedirects(body *conf.Block) {
	for _, d := range body.GetAll("redir") {
		m := body.Get(d.Arg(0))
		if m == nil || !strings.HasPrefix(d.Arg(0), "@ace_redirect_") {
			continue
		}
		r := redirectFromArgs(d.Arg(1), d.Arg(2))
		r.From = m.Arg(1)
		r.Type = types.RedirectTypeURL
		if m.Arg(0) == "host" {
			r.Type = types.RedirectTypeHost
		}
		v.redirects = append(v.redirects, r)
	}
	for _, h := range body.GetAll("handle_errors") {
		if d := h.Get("redir"); h.Arg(0) == "404" && d != nil {
			r := redirectFromArgs(d.Arg(0), d.Arg(1))
			r.Type = types.RedirectType404
			v.redirects = append(v.redirects, r)
		}
	}
}

func redirectFromArgs(to, code string) types.Redirect {
	status, _ := strconv.Atoi(code)
	return types.Redirect{
		To:         strings.TrimSuffix(to, "{uri}"),
		KeepURI:    strings.HasSuffix(to, "{uri}"),
		StatusCode: status,
	}
}
