package bootstrap

import (
	"github.com/acepanel/panel/internal/apps/apache"
	"github.com/acepanel/panel/internal/apps/codeserver"
	"github.com/acepanel/panel/internal/apps/docker"
	"github.com/acepanel/panel/internal/apps/fail2ban"
	"github.com/acepanel/panel/internal/apps/frp"
	"github.com/acepanel/panel/internal/apps/gitea"
	"github.com/acepanel/panel/internal/apps/mariadb"
	"github.com/acepanel/panel/internal/apps/memcached"
	"github.com/acepanel/panel/internal/apps/minio"
	"github.com/acepanel/panel/internal/apps/mysql"
	"github.com/acepanel/panel/internal/apps/nginx"
	"github.com/acepanel/panel/internal/apps/openresty"
	"github.com/acepanel/panel/internal/apps/percona"
	"github.com/acepanel/panel/internal/apps/phpmyadmin"
	"github.com/acepanel/panel/internal/apps/podman"
	"github.com/acepanel/panel/internal/apps/postgresql"
	"github.com/acepanel/panel/internal/apps/pureftpd"
	"github.com/acepanel/panel/internal/apps/redis"
	"github.com/acepanel/panel/internal/apps/rsync"
	"github.com/acepanel/panel/internal/apps/s3fs"
	"github.com/acepanel/panel/internal/apps/supervisor"
	"github.com/acepanel/panel/pkg/apploader"
)

func NewLoader(
	apache *apache.App,
	codeserver *codeserver.App,
	docker *docker.App,
	fail2ban *fail2ban.App,
	frp *frp.App,
	gitea *gitea.App,
	mariadb *mariadb.App,
	memcached *memcached.App,
	minio *minio.App,
	mysql *mysql.App,
	nginx *nginx.App,
	openresty *openresty.App,
	percona *percona.App,
	phpmyadmin *phpmyadmin.App,
	podman *podman.App,
	postgresql *postgresql.App,
	pureftpd *pureftpd.App,
	redis *redis.App,
	rsync *rsync.App,
	s3fs *s3fs.App,
	supervisor *supervisor.App,
) *apploader.Loader {
	loader := new(apploader.Loader)
	loader.Add(apache, codeserver, docker, fail2ban, frp, gitea, mariadb, memcached, minio, mysql, nginx, openresty, percona, phpmyadmin, podman, postgresql, pureftpd, redis, rsync, s3fs, supervisor)
	return loader
}
