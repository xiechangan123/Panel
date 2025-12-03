package apache

// DisablePagePath 禁用页面路径
const DisablePagePath = "/opt/ace/server/apache/stop"

// SitesPath 网站目录
const SitesPath = "/opt/ace/sites"

// 配置文件序号范围
const (
	RedirectStartNum = 100 // 重定向配置起始序号 (100-199)
	RedirectEndNum   = 199
	ProxyStartNum    = 200 // 代理配置起始序号 (200-299)
	ProxyEndNum      = 299
)

// DefaultVhostConf 默认配置模板
const DefaultVhostConf = `<VirtualHost *:80>
    ServerName localhost
    DocumentRoot /opt/ace/sites/default/public
    DirectoryIndex index.php index.html

    ErrorLog /opt/ace/sites/default/log/error.log
    CustomLog /opt/ace/sites/default/log/access.log combined

    # custom configs
    IncludeOptional /opt/ace/sites/default/config/site/*.conf

    <Directory /opt/ace/sites/default/public>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
</VirtualHost>
`
