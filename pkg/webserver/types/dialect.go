package types

// Features Web 服务器能力集，不支持的功能由调用方按需降级
type Features struct {
	IPv6Listen  bool // 站点额外监听 IPv6 地址
	Stat        bool // 访问统计
	DefaultSite bool // 默认站点切换
}

// Dialect 收敛某种 Web 服务器在面板层面的全部差异，新增服务器只需实现此接口并在 webserver 包中注册
type Dialect interface {
	// Service systemd 服务名
	Service() string
	// ConfigTest 配置测试命令
	ConfigTest() string
	// HTMLDir 默认页目录，存放 index.html、stop.html 与 404.html
	HTMLDir() string
	// PanelACMEConf 面板证书 HTTP 验证使用的独立配置文件
	PanelACMEConf() string
	// Features 能力集
	Features() Features
	// HTTPSListenArgs 443 监听的附加参数
	HTTPSListenArgs() []string
	// ErrorPageConf 站点 404 页面片段
	ErrorPageConf() string
	// PHPCacheConf PHP 站点浏览器缓存与敏感文件拦截片段
	PHPCacheConf() string
	// SPAConf 静态站点单页应用路由回退片段
	SPAConf() string
	// HTPasswdLine 基本认证 htpasswd 单行
	HTPasswdLine(username, password string) string

	NewStaticVhost(configDir string) (StaticVhost, error)
	NewPHPVhost(configDir string) (PHPVhost, error)
	NewProxyVhost(configDir string) (ProxyVhost, error)

	// WriteSiteChallenge 向网站 acme 配置文件投放一个 HTTP-01 验证
	WriteSiteChallenge(conf, path, token string) error
	// RemoveSiteChallenge 移除网站 acme 配置文件中的一个 HTTP-01 验证
	RemoveSiteChallenge(conf, path, token string) error
	// WritePanelChallenge 写入面板独立验证站点，用于 80 端口已被 Web 服务器占用时签发面板证书
	WritePanelChallenge(conf string, names []string, tokens map[string]string) error
	// RemovePanelChallenge 清理面板独立验证站点
	RemovePanelChallenge(conf string) error
}
