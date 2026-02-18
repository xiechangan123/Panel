package websitestat

import (
	"fmt"

	ua "github.com/medama-io/go-useragent"
)

var uaParser = ua.NewParser()

// ParseUA 解析 User-Agent，返回浏览器和操作系统名称
func ParseUA(rawUA string) (browser, os string) {
	agent := uaParser.Parse(rawUA)

	// 浏览器：名称 + 主版本号
	bName := string(agent.Browser())
	bMajor := agent.BrowserVersionMajor()
	if bName == "" {
		browser = "Other"
	} else if bMajor == "" {
		browser = bName
	} else {
		browser = fmt.Sprintf("%s %s", bName, bMajor)
	}

	// 操作系统
	osName := string(agent.OS())
	if osName == "" {
		os = "Other"
	} else {
		os = osName
	}

	return
}
