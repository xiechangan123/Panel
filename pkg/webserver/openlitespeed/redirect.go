package openlitespeed

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// URL 重定向写为 redirect 上下文，404 重定向写为 errorpage，主机名重定向写为 rewrite 规则

func redirectStatus(r types.Redirect) int {
	if r.StatusCode == 0 {
		return 308
	}
	return r.StatusCode
}

func (v *baseVhost) buildRedirects(cfg *conf.Config) {
	for _, r := range v.redirects {
		switch r.Type {
		case types.RedirectTypeURL:
			from := strings.TrimPrefix(r.From, "^")
			from = strings.TrimSuffix(strings.TrimSuffix(from, "(.*)$"), "$")
			uri, to := "exp:^"+from+"$", r.To
			if r.KeepURI {
				uri = "exp:^" + from + "(.*)$"
				if !strings.HasSuffix(to, "$1") {
					to += "$1"
				}
			}
			ctx := cfg.AddBlock("context", uri)
			ctx.Add("type", "redirect")
			ctx.Add("externalRedirect", "1")
			ctx.Add("statusCode", strconv.Itoa(redirectStatus(r)))
			ctx.Add("location", to)
			setHeaders(ctx, v.contextHeaders())
		case types.RedirectType404:
			cfg.AddBlock("errorpage", "404").Add("url", r.To)
		}
	}
}

var hostCondPattern = regexp.MustCompile(`^%\{HTTP_HOST\}\s+\^(.+)\$\s+\[NC\]$`)
var hostRulePattern = regexp.MustCompile(`^\^\(\.\*\)\$\s+(\S+)\s+\[R=(\d+),L\]$`)

func (v *baseVhost) loadRedirects(cfg *conf.Config) {
	for _, ctx := range cfg.Blocks("context") {
		if !strings.EqualFold(ctx.Value("type"), "redirect") {
			continue
		}
		from := strings.TrimPrefix(ctx.Arg(0), "exp:^")
		keepURI := strings.HasSuffix(from, "(.*)$")
		from = strings.TrimSuffix(strings.TrimSuffix(from, "(.*)$"), "$")
		code, _ := strconv.Atoi(ctx.Value("statusCode"))
		v.redirects = append(v.redirects, types.Redirect{
			Type:       types.RedirectTypeURL,
			From:       from,
			To:         strings.TrimSuffix(ctx.Value("location"), "$1"),
			KeepURI:    keepURI,
			StatusCode: code,
		})
	}

	for _, page := range cfg.Blocks("errorpage") {
		if page.Arg(0) == "404" {
			v.redirects = append(v.redirects, types.Redirect{
				Type:       types.RedirectType404,
				To:         page.Value("url"),
				StatusCode: 308,
			})
		}
	}

	rw := cfg.GetBlock("rewrite")
	if rw == nil {
		return
	}
	host := ""
	for _, n := range rw.Nodes {
		d, ok := n.(*conf.Directive)
		if !ok {
			continue
		}
		switch {
		case strings.EqualFold(d.Name, "RewriteCond"):
			host = ""
			if m := hostCondPattern.FindStringSubmatch(d.Arg(0)); m != nil {
				host = strings.ReplaceAll(m[1], `\.`, ".")
			}
		case strings.EqualFold(d.Name, "RewriteRule") && host != "":
			if m := hostRulePattern.FindStringSubmatch(d.Arg(0)); m != nil {
				code, _ := strconv.Atoi(m[2])
				v.redirects = append(v.redirects, types.Redirect{
					Type:       types.RedirectTypeHost,
					From:       host,
					To:         strings.TrimSuffix(m[1], "$1"),
					KeepURI:    strings.HasSuffix(m[1], "$1"),
					StatusCode: code,
				})
			}
			host = ""
		}
	}
}
