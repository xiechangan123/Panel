package nginx

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

var upstreamFilePattern = regexp.MustCompile(`^(\d{3})-(.+)\.conf$`)

var upstreamAlgos = []string{"least_conn", "ip_hash", "hash", "random"}

func parseUpstreamFiles(sharedDir string) []types.Upstream {
	var upstreams []types.Upstream
	for _, file := range listFiles(sharedDir, upstreamFilePattern, UpstreamStartNum, math.MaxInt) {
		cfg, err := ParseFile(file)
		if err != nil {
			continue
		}
		up := cfg.GetBlock("upstream")
		if up == nil || up.Arg(0) == "" {
			continue
		}
		upstream := types.Upstream{
			Name:      up.Arg(0),
			Servers:   make(map[string]string),
			Resolver:  []string{},
			Keepalive: atoi(up.Value("keepalive")),
		}
		for _, algo := range upstreamAlgos {
			if up.Has(algo) {
				upstream.Algo = algo
				break
			}
		}
		for _, srv := range up.GetAll("server") {
			if srv.Arg(0) != "" {
				upstream.Servers[srv.Arg(0)] = strings.Join(srv.Values()[1:], " ")
			}
		}
		if resolver := up.Get("resolver").Values(); resolver != nil {
			upstream.Resolver = resolver
		}
		upstream.ResolverTimeout = parseDuration(up.Value("resolver_timeout"))
		upstreams = append(upstreams, upstream)
	}
	return upstreams
}

func writeUpstreamFiles(sharedDir string, upstreams []types.Upstream) error {
	if err := clearFiles(sharedDir, upstreamFilePattern, UpstreamStartNum, math.MaxInt); err != nil {
		return err
	}
	for i, upstream := range upstreams {
		path := filepath.Join(sharedDir, fmt.Sprintf("%03d-%s.conf", UpstreamStartNum+i, upstream.Name))
		if err := writeFragment(path, conf.Cmt("Upstream: "+upstream.Name), upstreamNode(upstream)); err != nil {
			return err
		}
	}
	return nil
}

func clearUpstreamFiles(sharedDir string) error {
	return clearFiles(sharedDir, upstreamFilePattern, UpstreamStartNum, math.MaxInt)
}

func upstreamNode(u types.Upstream) *conf.Directive {
	up := conf.Blk("upstream", u.Name)
	up.Add("zone", u.Name, "512k")
	if u.Algo != "" {
		up.Add(u.Algo)
	}
	if len(u.Resolver) > 0 {
		up.Add("resolver", u.Resolver...)
		if u.ResolverTimeout > 0 {
			up.Add("resolver_timeout", formatDuration(u.ResolverTimeout))
		}
	}
	for _, addr := range sortedKeys(u.Servers) {
		up.Add("server", append([]string{addr}, strings.Fields(u.Servers[addr])...)...)
	}
	if u.Keepalive > 0 {
		up.Add("keepalive", strconv.Itoa(u.Keepalive))
	}
	return up
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
