package charset

import (
	"errors"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{name: "空文件", data: nil, want: UTF8},
		{name: "ascii", data: []byte("hello"), want: UTF8},
		{name: "utf-8", data: []byte("中文"), want: UTF8},
		{name: "utf-8 bom", data: []byte{0xEF, 0xBB, 0xBF, 0xE4, 0xB8, 0xAD}, want: "utf-8-bom"},
		{name: "utf-16le bom", data: []byte{0xFF, 0xFE, 0x2D, 0x4E}, want: "utf-16le"},
		{name: "utf-16be bom", data: []byte{0xFE, 0xFF, 0x4E, 0x2D}, want: "utf-16be"},
		{name: "gbk 回退", data: []byte{0xD6, 0xD0, 0xCE, 0xC4}, want: GB18030},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			check.Equal(t, Detect(test.data), test.want)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		encoding string
		text     string
	}{
		{encoding: UTF8, text: "中文 abc"},
		{encoding: "utf-8-bom", text: "中文 abc"},
		{encoding: "utf-16le", text: "中文 abc"},
		{encoding: "utf-16be", text: "中文 abc"},
		{encoding: GB18030, text: "中文 abc 😀"},
		{encoding: "gbk", text: "中文 abc"},
		{encoding: "big5", text: "中文 abc"},
		{encoding: "shift_jis", text: "日本語 abc"},
		{encoding: "euc-jp", text: "日本語 abc"},
		{encoding: "iso-2022-jp", text: "日本語 abc"},
		{encoding: "euc-kr", text: "한국어 abc"},
		{encoding: "windows-1252", text: "café abc"},
		{encoding: "windows-1251", text: "привет abc"},
	}
	for _, test := range tests {
		t.Run(test.encoding, func(t *testing.T) {
			encoded, err := Encode([]byte(test.text), test.encoding)
			must.NoError(t, err)

			decoded, err := Decode(encoded, test.encoding)
			must.NoError(t, err)
			check.Equal(t, string(decoded), test.text)
		})
	}
}

// GB18030 向下兼容 GBK
func TestGB18030CompatibleWithGBK(t *testing.T) {
	text := []byte("中文简体 GB2312 区 abc")

	gbk, err := Encode(text, "gbk")
	must.NoError(t, err)
	gb18030, err := Encode(text, GB18030)
	must.NoError(t, err)
	check.DeepEqual(t, gb18030, gbk)

	decoded, err := Decode(gbk, GB18030)
	must.NoError(t, err)
	check.DeepEqual(t, decoded, text)
}

func TestBOM(t *testing.T) {
	// 编码时写入 BOM
	encoded, err := Encode([]byte("中"), "utf-8-bom")
	must.NoError(t, err)
	check.DeepEqual(t, encoded, []byte{0xEF, 0xBB, 0xBF, 0xE4, 0xB8, 0xAD})

	// 解码时剥离 BOM
	decoded, err := Decode(encoded, "utf-8-bom")
	must.NoError(t, err)
	check.Equal(t, string(decoded), "中")

	// UTF-16 编码需带 BOM，否则字节序无从判断
	encoded, err = Encode([]byte("中"), "utf-16le")
	must.NoError(t, err)
	check.DeepEqual(t, encoded, []byte{0xFF, 0xFE, 0x2D, 0x4E})
}

func TestEncodeUnsupportedRune(t *testing.T) {
	_, err := Encode([]byte("第一行\n第二行 😀"), "gbk")
	must.Error(t, err)

	var unsupported *UnsupportedRuneError
	ok := errors.As(err, &unsupported)
	must.True(t, ok)
	check.Equal(t, unsupported.Rune, '😀')
	check.Equal(t, unsupported.Line, 2)
}

func TestUnsupportedEncoding(t *testing.T) {
	_, err := Encode([]byte("abc"), "replacement")
	check.Error(t, err)
	_, err = Decode([]byte("abc"), "replacement")
	check.Error(t, err)
}

// 编码名为空时原样透传，图片等二进制内容依赖该行为
func TestPassthrough(t *testing.T) {
	raw := []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0xFF}

	decoded, err := Decode(raw, "")
	must.NoError(t, err)
	check.DeepEqual(t, decoded, raw)

	encoded, err := Encode(raw, "")
	must.NoError(t, err)
	check.DeepEqual(t, encoded, raw)
}
