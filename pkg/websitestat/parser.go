package websitestat

import (
	"encoding/json"
)

// ParseSyslog 从 syslog 消息中提取 tag 和 JSON 体
// 格式: <PRI>MMM DD HH:MM:SS tag: JSON（nohostname 模式下无主机名）
func ParseSyslog(msg []byte) (string, []byte) {
	n := len(msg)
	if n == 0 {
		return "", nil
	}

	i := 0
	// 跳过 PRI: <xxx>
	if msg[0] == '<' {
		for i < n && msg[i] != '>' {
			i++
		}
		if i < n {
			i++ // 跳过 '>'
		}
	}

	// 找到 ": " 分隔符，tag 在其前面的最后一个空格之后
	colonIdx := -1
	for j := i; j < n-1; j++ {
		if msg[j] == ':' && msg[j+1] == ' ' {
			colonIdx = j
			break
		}
	}
	if colonIdx < 0 {
		return "", nil
	}

	// tag 是 ": " 前面最后一个空格之后的部分
	tagStart := i
	for j := colonIdx - 1; j >= i; j-- {
		if msg[j] == ' ' {
			tagStart = j + 1
			break
		}
	}

	tag := string(msg[tagStart:colonIdx])
	data := msg[colonIdx+2:] // 跳过 ": "

	return tag, data
}

// jsonEntry 内部 JSON 解析结构（与 nginx log_format 对应）
type jsonEntry struct {
	URI         string      `json:"uri"`
	Status      json.Number `json:"status"`
	Bytes       json.Number `json:"bytes"`
	UA          string      `json:"ua"`
	IP          string      `json:"ip"`
	Method      string      `json:"method"`
	Body        string      `json:"body"`
	ContentType string      `json:"content_type"`
	ReqLength   json.Number `json:"req_length"`
}

// ParseLogEntry 将 JSON 数据解析为 LogEntry
func ParseLogEntry(tag string, data []byte) (*LogEntry, error) {
	var je jsonEntry
	if err := json.Unmarshal(data, &je); err != nil {
		return nil, err
	}

	status, _ := je.Status.Int64()
	bytes, _ := je.Bytes.Int64()
	reqLen, _ := je.ReqLength.Int64()

	entry := &LogEntry{
		Site:        tag,
		URI:         je.URI,
		Status:      int(status),
		Bytes:       uint64(bytes),
		UA:          je.UA,
		IP:          je.IP,
		Method:      je.Method,
		ContentType: je.ContentType,
		ReqLength:   uint64(reqLen),
	}

	// 仅在 4xx/5xx 且请求体 <= 64KB 时保留 body
	if status >= 400 && status < 600 && len(je.Body) <= 65536 {
		entry.Body = je.Body
	}

	return entry, nil
}
