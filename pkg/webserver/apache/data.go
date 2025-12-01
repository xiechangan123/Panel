package apache

// DisableConfName 禁用配置文件名
const DisableConfName = "00-disable.conf"

// DisableConfContent 禁用配置内容
const DisableConfContent = `# 网站已停止
RewriteEngine on
RewriteRule ^.*$ - [R=503,L]
`

// DefaultVhostConf 默认配置模板
const DefaultVhostConf = `<VirtualHost *:80>
    ServerName localhost
    DocumentRoot /opt/ace/sites/default/public
    DirectoryIndex index.php index.html

    ErrorLog /opt/ace/sites/default/log/error.log
    CustomLog /opt/ace/sites/default/log/access.log combined

    # custom configs
    IncludeOptional /opt/ace/sites/default/config/server.d/*.conf

    <Directory /opt/ace/sites/default/public>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
</VirtualHost>
`
