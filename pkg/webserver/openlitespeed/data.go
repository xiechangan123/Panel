package openlitespeed

// ServerRoot 安装根目录
const ServerRoot = "/opt/ace/server/openlitespeed"

// HTMLDir 默认页目录
const HTMLDir = ServerRoot + "/html"

// DisablePage 禁用页面路径
const DisablePage = HTMLDir + "/stop.html"

// PanelConfDir 面板托管的主配置片段目录，由主配置 include
const PanelConfDir = ServerRoot + "/conf/panel"

// ListenersConf 面板生成的监听器配置
const ListenersConf = PanelConfDir + "/listeners.conf"

// SitesPath 网站目录
const SitesPath = "/opt/ace/sites"

// 站点配置目录下的文件名
const (
	VhostConfName    = "openlitespeed.conf"        // 站点 vhconf，由面板生成
	RegisterConfName = "openlitespeed.vhost.conf"  // 主配置中的站点注册片段
	ListenConfName   = "openlitespeed.listen.conf" // 面板记录的监听地址与域名
)

// stopURI 站点停用时所有请求重写到的路径，由静态上下文映射到默认页目录
const stopURI = "/ace-stop/"

// acmeURI HTTP-01 验证路径
const acmeURI = "/.well-known/acme-challenge/"

// accessLogFormat Apache 兼容的访问日志格式
const accessLogFormat = `"%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\""`

const phpCacheConf = `expires {
  enableExpires           1
  expiresByType           image/*=A2592000,text/css=A21600,application/javascript=A21600,application/x-javascript=A21600,font/*=A21600,application/font-woff=A21600,application/font-woff2=A21600,application/vnd.ms-fontobject=A21600
}
# deny sensitive files
context exp:^/\.(user\.ini|htaccess|git|svn|env) {
  location                $DOC_ROOT/
  allowBrowse             0
}
`

const spaConf = `# single-page application route fallback, remove if not needed
RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule ^ /index.html [L]
`

const errorPageConf = `errorpage 404 {
  url                     /404.html
}
`
