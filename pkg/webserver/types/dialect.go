package types

// Features Web 服务器能力集，不支持的功能由调用方按需降级
type Features struct {
	IPv6Listen  bool // 站点额外监听 IPv6 地址
	Stat        bool // 访问统计
	DefaultSite bool // 默认站点切换
	LSCache     bool // LiteSpeed 页面缓存
}

// Dialect 收敛某种 Web 服务器在面板层面的全部差异，新增服务器只需实现此接口并在 webserver 包中注册
type Dialect interface {
	// Service systemd 服务名
	Service() string
	// ConfigTest 配置测试命令
	ConfigTest() string
	// HTMLDir 默认页目录，存放 index.html、stop.html 与 404.html
	HTMLDir() string
	// ConfigFile 站点配置目录中的主配置文件名
	ConfigFile() string
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
	// LSCacheConf 站点级 LiteSpeed 页面缓存片段，name 为站点名
	LSCacheConf(name string) string
	// StatConf 访问统计片段，shared 为共享级、site 为站点级，不支持统计时都为空
	StatConf(name string) (shared, site string)
	// DefaultSiteConf 内置默认站点的独立配置文件，没有时为空
	DefaultSiteConf() string
	// WriteDefaultSite 写入内置默认站点配置，asDefault 为 false 时把默认位让给某个站点
	WriteDefaultSite(asDefault bool) error
	// HTPasswdLine 基本认证 htpasswd 单行
	HTPasswdLine(username, password string) string
	// RewritesDir 伪静态预置目录名，语法相同的服务器可共用
	RewritesDir() string
	// BeforeReload 重载前的准备工作，如重建全局配置
	BeforeReload() error

	NewStaticVhost(configDir string) (StaticVhost, error)
	NewPHPVhost(configDir string) (PHPVhost, error)
	NewProxyVhost(configDir string) (ProxyVhost, error)

	// 以下方法的 bool 返回值表示配置是否变化，未变化则跳过重载
	// WriteSiteChallenge 向网站 acme 配置文件投放一个 HTTP-01 验证
	WriteSiteChallenge(conf, path, token string) (bool, error)
	// RemoveSiteChallenge 移除网站 acme 配置文件中的一个 HTTP-01 验证
	RemoveSiteChallenge(conf, path, token string) (bool, error)
	// WritePanelChallenge 写入面板独立验证站点，用于 80 端口已被 Web 服务器占用时签发面板证书
	WritePanelChallenge(conf string, names []string, tokens map[string]string) (bool, error)
	// RemovePanelChallenge 清理面板独立验证站点
	RemovePanelChallenge(conf string) (bool, error)
}
