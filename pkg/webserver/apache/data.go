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
    # custom configs
    IncludeOptional /opt/ace/sites/default/config/site/*.conf
    <Directory /opt/ace/sites/default/public>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
</VirtualHost>
`

// order 定义 Apache 指令的排序优先级
var order = map[string]int{
	"Listen":     0,
	"ServerName": 1,

	"ServerAlias": 10,
	"ServerAdmin": 11,

	"DocumentRoot":   100,
	"DirectoryIndex": 101,
	"Options":        102,
	"AllowOverride":  103,
	"Require":        104,
	"Order":          105,
	"Allow":          106,
	"Deny":           107,

	"LimitRequestBody":      200,
	"LimitRequestFields":    201,
	"LimitRequestFieldSize": 202,
	"LimitRequestLine":      203,
	"LimitXMLRequestBody":   204,

	"AuthType":          300,
	"AuthName":          301,
	"AuthUserFile":      302,
	"AuthGroupFile":     303,
	"AuthBasicProvider": 304,

	"SSLEngine":                        400,
	"SSLCertificateFile":               401,
	"SSLCertificateKeyFile":            402,
	"SSLCertificateChainFile":          403,
	"SSLCACertificateFile":             404,
	"SSLCACertificatePath":             405,
	"SSLProtocol":                      406,
	"SSLCipherSuite":                   407,
	"SSLHonorCipherOrder":              408,
	"SSLCompression":                   409,
	"SSLSessionCache":                  410,
	"SSLSessionCacheTimeout":           411,
	"SSLSessionTickets":                412,
	"SSLUseStapling":                   413,
	"SSLStaplingCache":                 414,
	"SSLStaplingResponderTimeout":      415,
	"SSLStaplingReturnResponderErrors": 416,
	"SSLInsecureRenegotiation":         417,
	"SSLVerifyClient":                  418,
	"SSLVerifyDepth":                   419,
	"SSLOptions":                       420,

	"Header":          500,
	"RequestHeader":   501,
	"SetEnvIf":        502,
	"SetEnvIfNoCase":  503,
	"SetEnv":          504,
	"UnsetEnv":        505,
	"PassEnv":         506,
	"SetOutputFilter": 507,
	"SetInputFilter":  508,
	"AddOutputFilter": 509,
	"AddInputFilter":  510,
	"AddType":         511,
	"AddHandler":      512,
	"AddCharset":      513,
	"AddEncoding":     514,
	"AddLanguage":     515,
	"DefaultType":     516,
	"ForceType":       517,
	"RemoveType":      518,
	"RemoveHandler":   519,
	"RemoveCharset":   520,
	"RemoveEncoding":  521,
	"RemoveLanguage":  522,

	"ProxyPass":                    600,
	"ProxyPassReverse":             601,
	"ProxyPassMatch":               602,
	"ProxyPassReverseCookieDomain": 603,
	"ProxyPassReverseCookiePath":   604,
	"ProxyPreserveHost":            605,
	"ProxyRequests":                606,
	"ProxyVia":                     607,
	"ProxyTimeout":                 608,
	"ProxyAddHeaders":              609,
	"ProxySet":                     610,
	"BalancerMember":               611,
	"ProxyPassInherit":             612,
	"ProxyPassInterpolateEnv":      613,

	"RewriteEngine":  700,
	"RewriteBase":    701,
	"RewriteCond":    702,
	"RewriteRule":    703,
	"RewriteMap":     704,
	"RewriteOptions": 705,

	"Redirect":          800,
	"RedirectMatch":     801,
	"RedirectTemp":      802,
	"RedirectPermanent": 803,

	"Alias":            900,
	"AliasMatch":       901,
	"ScriptAlias":      902,
	"ScriptAliasMatch": 903,

	"ErrorDocument": 1000,

	"ExpiresActive":           1100,
	"ExpiresDefault":          1101,
	"ExpiresByType":           1102,
	"DeflateCompressionLevel": 1103,
	"DeflateMemLevel":         1104,
	"DeflateWindowSize":       1105,
	"DeflateBufferSize":       1106,
	"DeflateFilterNote":       1107,
	"AddOutputFilterByType":   1108,

	"PHPIniDir":  1200,
	"SetHandler": 1201,

	"Directory":      1300,
	"DirectoryMatch": 1301,
	"Files":          1302,
	"FilesMatch":     1303,
	"Location":       1304,
	"LocationMatch":  1305,
	"If":             1306,
	"IfDefine":       1307,
	"IfModule":       1308,
	"Else":           1309,
	"ElseIf":         1310,
	"Proxy":          1311,
	"ProxyMatch":     1312,

	"Include":         1290,
	"IncludeOptional": 1291,

	"ErrorLog":    1500,
	"CustomLog":   1501,
	"LogLevel":    1502,
	"LogFormat":   1503,
	"TransferLog": 1504,
}
