package websitestat

import "strings"

// 已知搜索引擎蜘蛛 UA 关键词
var spiderKeywords = []string{
	"googlebot",
	"bingbot",
	"baiduspider",
	"yandexbot",
	"sogou",
	"360spider",
	"bytespider",
	"gptbot",
	"claudebot",
	"ahrefsbot",
	"semrushbot",
	"dotbot",
	"mj12bot",
	"petalbot",
	"applebot",
	"duckduckbot",
	"slurp",
	"ia_archiver",
	"facebookexternalhit",
	"twitterbot",
	"rogerbot",
	"linkedinbot",
	"embedly",
	"quora link preview",
	"showyoubot",
	"outbrain",
	"pinterest",
	"slackbot",
	"vkshare",
	"w3c_validator",
	"redditbot",
	"scrapy",
	"curl",
	"wget",
}

// IsSpider 检测 User-Agent 是否为已知蜘蛛
func IsSpider(ua string) bool {
	lower := strings.ToLower(ua)
	for _, kw := range spiderKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
