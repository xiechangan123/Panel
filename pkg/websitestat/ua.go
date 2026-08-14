package websitestat

import (
	ua "github.com/medama-io/go-useragent"
)

var uaParser = ua.NewParser()

// ParseUA 解析 User-Agent，返回浏览器和操作系统名称
func ParseUA(rawUA string) (browser, os string) {
	agent := uaParser.Parse(rawUA)

	// 浏览器：名称 + 主版本号
	bName := string(agent.Browser())
	bMajor := agent.BrowserVersionMajor()
	switch {
	case bName == "":
		browser = "Other"
	case bMajor == "":
		browser = bName
	default:
		browser = bName + " " + bMajor
	}

	// 操作系统
	os = "Other"
	if osName := string(agent.OS()); osName != "" {
		os = osName
	}

	return
}
