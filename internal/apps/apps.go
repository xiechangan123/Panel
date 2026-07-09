package apps

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
)

var Package = do.Package(
	do.Lazy(apache.NewApp), do.Lazy(clickhouse.NewApp), do.Lazy(codeserver.NewApp),
	do.Lazy(docker.NewApp), do.Lazy(elasticsearch.NewApp), do.Lazy(fail2ban.NewApp),
	do.Lazy(frp.NewApp), do.Lazy(gitea.NewApp), do.Lazy(grafana.NewApp),
	do.Lazy(kafka.NewApp), do.Lazy(mariadb.NewApp), do.Lazy(memcached.NewApp),
	do.Lazy(minio.NewApp), do.Lazy(mongodb.NewApp), do.Lazy(mysql.NewApp),
	do.Lazy(nginx.NewApp), do.Lazy(openresty.NewApp), do.Lazy(opensearch.NewApp),
	do.Lazy(percona.NewApp), do.Lazy(phpmyadmin.NewApp), do.Lazy(podman.NewApp),
	do.Lazy(postgresql.NewApp), do.Lazy(prometheus.NewApp), do.Lazy(pureftpd.NewApp),
	do.Lazy(redis.NewApp), do.Lazy(rocketmq.NewApp), do.Lazy(rsync.NewApp),
	do.Lazy(s3fs.NewApp), do.Lazy(supervisor.NewApp), do.Lazy(valkey.NewApp),
)
