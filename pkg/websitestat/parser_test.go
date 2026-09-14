package websitestat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSyslogNginx(t *testing.T) {
	tag, data := ParseSyslog([]byte(`<190>Sep 14 10:00:00 ace_stat_demo: {"site":"demo","uri":"/a","status":200,"bytes":10,"ua":"curl","ip":"1.2.3.4","method":"GET","content_type":"text/html","req_length":100,"rt":0.5}`))
	require.Equal(t, "ace_stat_demo", tag)
	entry, err := ParseLogEntry(tag, data)
	require.NoError(t, err)
	require.Equal(t, &LogEntry{Site: "demo", URI: "/a", Status: 200, Bytes: 10, UA: "curl", IP: "1.2.3.4", Method: "GET", ContentType: "text/html", ReqLength: 100, RequestTime: 0.5}, entry)
}

func TestParseLogEntryCaddy(t *testing.T) {
	msg := []byte(`{"level":"info","ts":1789386806.38,"logger":"http.log.access.ace_stat","msg":"handled request","request":{"remote_ip":"::1","remote_port":"49367","client_ip":"::1","proto":"HTTP/2.0","method":"POST","host":"localhost","uri":"/api?x=1","headers":{"User-Agent":["curl/8.7.1"],"Accept":["*/*"]}},"bytes_read":42,"user_id":"","duration":0.0009,"size":11,"status":404,"resp_headers":{"Content-Type":["text/html; charset=utf-8"]},"site":"demo"}`)
	tag, data := ParseSyslog(msg)
	require.Equal(t, "", tag)
	entry, err := ParseLogEntry(tag, data)
	require.NoError(t, err)
	require.Equal(t, &LogEntry{Site: "demo", URI: "/api?x=1", Status: 404, Bytes: 11, UA: "curl/8.7.1", IP: "::1", Method: "POST", ContentType: "text/html; charset=utf-8", ReqLength: 42, RequestTime: 0.0009}, entry)
}
