package bootstrap

import (
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/apps/apache"
	"github.com/acepanel/panel/v3/internal/apps/clickhouse"
	"github.com/acepanel/panel/v3/internal/apps/codeserver"
	"github.com/acepanel/panel/v3/internal/apps/docker"
	"github.com/acepanel/panel/v3/internal/apps/elasticsearch"
	"github.com/acepanel/panel/v3/internal/apps/fail2ban"
	"github.com/acepanel/panel/v3/internal/apps/frp"
	"github.com/acepanel/panel/v3/internal/apps/gitea"
	"github.com/acepanel/panel/v3/internal/apps/grafana"
	"github.com/acepanel/panel/v3/internal/apps/kafka"
	"github.com/acepanel/panel/v3/internal/apps/mariadb"
	"github.com/acepanel/panel/v3/internal/apps/memcached"
	"github.com/acepanel/panel/v3/internal/apps/minio"
	"github.com/acepanel/panel/v3/internal/apps/mongodb"
	"github.com/acepanel/panel/v3/internal/apps/mysql"
	"github.com/acepanel/panel/v3/internal/apps/nginx"
	"github.com/acepanel/panel/v3/internal/apps/openresty"
	"github.com/acepanel/panel/v3/internal/apps/opensearch"
	"github.com/acepanel/panel/v3/internal/apps/percona"
	"github.com/acepanel/panel/v3/internal/apps/phpmyadmin"
	"github.com/acepanel/panel/v3/internal/apps/podman"
	"github.com/acepanel/panel/v3/internal/apps/postgresql"
	"github.com/acepanel/panel/v3/internal/apps/prometheus"
	"github.com/acepanel/panel/v3/internal/apps/pureftpd"
	"github.com/acepanel/panel/v3/internal/apps/redis"
	"github.com/acepanel/panel/v3/internal/apps/rocketmq"
	"github.com/acepanel/panel/v3/internal/apps/rsync"
	"github.com/acepanel/panel/v3/internal/apps/s3fs"
	"github.com/acepanel/panel/v3/internal/apps/supervisor"
	"github.com/acepanel/panel/v3/internal/apps/valkey"
	"github.com/acepanel/panel/v3/pkg/apploader"
)

func NewLoader(i do.Injector) (*apploader.Loader, error) {
	loader := new(apploader.Loader)
	loader.Add(
		do.MustInvoke[*apache.App](i),
		do.MustInvoke[*clickhouse.App](i),
		do.MustInvoke[*codeserver.App](i),
		do.MustInvoke[*docker.App](i),
		do.MustInvoke[*elasticsearch.App](i),
		do.MustInvoke[*fail2ban.App](i),
		do.MustInvoke[*frp.App](i),
		do.MustInvoke[*gitea.App](i),
		do.MustInvoke[*grafana.App](i),
		do.MustInvoke[*kafka.App](i),
		do.MustInvoke[*mariadb.App](i),
		do.MustInvoke[*memcached.App](i),
		do.MustInvoke[*minio.App](i),
		do.MustInvoke[*mongodb.App](i),
		do.MustInvoke[*mysql.App](i),
		do.MustInvoke[*nginx.App](i),
		do.MustInvoke[*openresty.App](i),
		do.MustInvoke[*opensearch.App](i),
		do.MustInvoke[*percona.App](i),
		do.MustInvoke[*phpmyadmin.App](i),
		do.MustInvoke[*podman.App](i),
		do.MustInvoke[*postgresql.App](i),
		do.MustInvoke[*prometheus.App](i),
		do.MustInvoke[*pureftpd.App](i),
		do.MustInvoke[*redis.App](i),
		do.MustInvoke[*rocketmq.App](i),
		do.MustInvoke[*rsync.App](i),
		do.MustInvoke[*s3fs.App](i),
		do.MustInvoke[*supervisor.App](i),
		do.MustInvoke[*valkey.App](i),
	)
	return loader, nil
}
