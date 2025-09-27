package nginx

var order = []string{"listen", "server_name", "index", "root",
	"ssl_certificate", "ssl_certificate_key", "ssl_session_timeout", "ssl_session_cache", "ssl_protocols", "ssl_ciphers", "ssl_prefer_server_ciphers", "ssl_early_data", "ssl_stapling", "ssl_stapling_verify", "ssl_trusted_certificate",
	"resolver", "error_page", "include", "if", "location", "add_header", "access_log", "error_log"}

const DefaultConf = `server {
    listen 80;
    server_name localhost;
    index index.php index.html;
    root /www/wwwroot/default;
    # Error page
    error_page 404 /404.html;
    include enable-php-0.conf;
    # Browser cache
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
    # Deny sensitive files
    location ~ ^/(\.user.ini|\.htaccess|\.git|\.svn|\.env) {
        return 404;
    }
    access_log /www/wwwlogs/default.log;
    error_log /www/wwwlogs/default.log;
}
`
