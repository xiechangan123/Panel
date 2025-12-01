package nginx

// DisableConfName 禁用配置文件名
const DisableConfName = "00-disable.conf"

// DisableConfContent 禁用配置内容
const DisableConfContent = `# 网站已停止
location / {
    return 503;
}
`

const DefaultConf = `include /opt/ace/sites/default/config/http.d/*.conf;
server {
    listen 80;
    server_name localhost;
    index index.php index.html;
    root /opt/ace/sites/default/public;
    # error page
    error_page 404 /404.html;
    # custom configs
    include /opt/ace/sites/default/config/server.d/*.conf;
    # browser cache
    location ~ .*\.(bmp|jpg|jpeg|png|gif|svg|ico|tiff|webp|avif|heif|heic|jxl)$ {
        expires 30d;
        access_log /dev/null;
        error_log /dev/null;
    }
    location ~ .*\.(js|css|ttf|otf|woff|woff2|eot)$ {
        expires 6h;
        access_log /dev/null;
        error_log /dev/null;
    }
    # deny sensitive files
    location ~ ^/(\.user.ini|\.htaccess|\.git|\.svn|\.env) {
        return 404;
    }
    access_log /opt/ace/sites/default/log/access.log;
    error_log /opt/ace/sites/default/log/error.log;
}
`
