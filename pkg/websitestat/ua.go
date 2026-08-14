package websitestat

import (
	ua "github.com/medama-io/go-useragent"
)

var uaParser = ua.NewParser()

// ClassifyUA 单次解析 UA
func ClassifyUA(rawUA string) (spider, browser, os string) {
	if spider = spiderByKeyword(rawUA); spider != "" {
		return spider, "", ""
	}

	agent := uaParser.Parse(rawUA)
	if agent.IsBot() {
		return "Other", "", ""
	}

	return "", browserOf(agent), osOf(agent)
}

// ParseUA 解析 User-Agent
func ParseUA(rawUA string) (browser, os string) {
	agent := uaParser.Parse(rawUA)
	return browserOf(agent), osOf(agent)
}

// browserOf 浏览器名称 + 主版本号
func browserOf(agent ua.UserAgent) string {
	name := string(agent.Browser())
	major := agent.BrowserVersionMajor()
	switch {
	case name == "":
		return "Other"
	case major == "":
		return name
	default:
		return name + " " + major
	}
}

func osOf(agent ua.UserAgent) string {
	if name := string(agent.OS()); name != "" {
		return name
	}
	return "Other"
}
