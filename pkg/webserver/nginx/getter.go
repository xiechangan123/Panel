package nginx

import (
	"fmt"
	"slices"
	"strings"
)

func (p *Parser) GetRoot() (string, error) {
	directive, err := p.FindOne("server.root")
	if err != nil {
		return "", err
	}
	if len(p.parameters2Slices(directive.GetParameters())) == 0 {
		return "", nil
	}

	return directive.GetParameters()[0].GetValue(), nil
}

func (p *Parser) GetRootWithComment() (string, []string, error) {
	directive, err := p.FindOne("server.root")
	if err != nil {
		return "", nil, err
	}
	if len(p.parameters2Slices(directive.GetParameters())) == 0 {
		return "", directive.GetComment(), nil
	}

	return directive.GetParameters()[0].GetValue(), directive.GetComment(), nil
}

func (p *Parser) GetIncludes() (includes []string, comments [][]string, err error) {
	directives, err := p.Find("server.include")
	if err != nil {
		return nil, nil, err
	}

	for _, dir := range directives {
		if len(dir.GetParameters()) != 1 {
			return nil, nil, fmt.Errorf("invalid include directive, expected 1 parameter but got %d", len(dir.GetParameters()))
		}
		includes = append(includes, dir.GetParameters()[0].GetValue())
		comments = append(comments, dir.GetComment())
	}

	return includes, comments, nil
}

func (p *Parser) GetHTTPSProtocols() []string {
	directive, err := p.FindOne("server.ssl_protocols")
	if err != nil {
		return nil
	}

	return p.parameters2Slices(directive.GetParameters())
}

func (p *Parser) GetHTTPSCiphers() string {
	directive, err := p.FindOne("server.ssl_ciphers")
	if err != nil {
		return ""
	}
	if len(p.parameters2Slices(directive.GetParameters())) == 0 {
		return ""
	}

	return directive.GetParameters()[0].GetValue()
}

func (p *Parser) GetOCSP() bool {
	directive, err := p.FindOne("server.ssl_stapling")
	if err != nil {
		return false
	}
	if len(p.parameters2Slices(directive.GetParameters())) == 0 {
		return false
	}

	return directive.GetParameters()[0].GetValue() == "on"
}

func (p *Parser) GetHSTS() bool {
	directives, err := p.Find("server.add_header")
	if err != nil {
		return false
	}

	for _, dir := range directives {
		if slices.Contains(p.parameters2Slices(dir.GetParameters()), "Strict-Transport-Security") {
			return true
		}
	}

	return false
}

func (p *Parser) GetHTTPSRedirect() bool {
	directives, err := p.Find("server.if")
	if err != nil {
		return false
	}

	for _, dir := range directives {
		for _, dir2 := range dir.GetBlock().GetDirectives() {
			if dir2.GetName() == "return" && slices.Contains(p.parameters2Slices(dir2.GetParameters()), "https://$host$request_uri") {
				return true
			}
		}
	}

	return false
}

func (p *Parser) GetAltSvc() string {
	directive, err := p.FindOne("server.add_header")
	if err != nil {
		return ""
	}

	for i, param := range p.parameters2Slices(directive.GetParameters()) {
		if strings.HasPrefix(param, "Alt-Svc") && i+1 < len(p.parameters2Slices(directive.GetParameters())) {
			return p.parameters2Slices(directive.GetParameters())[i+1]
		}
	}

	return ""
}

func (p *Parser) GetAccessLog() (string, error) {
	directive, err := p.FindOne("server.access_log")
	if err != nil {
		return "", err
	}
	if len(p.parameters2Slices(directive.GetParameters())) == 0 {
		return "", nil
	}

	return directive.GetParameters()[0].GetValue(), nil
}

func (p *Parser) GetErrorLog() (string, error) {
	directive, err := p.FindOne("server.error_log")
	if err != nil {
		return "", err
	}
	if len(p.parameters2Slices(directive.GetParameters())) == 0 {
		return "", nil
	}

	return directive.GetParameters()[0].GetValue(), nil
}

// GetLimitRate 获取限速配置
func (p *Parser) GetLimitRate() string {
	directive, err := p.FindOne("server.limit_rate")
	if err != nil {
		return ""
	}
	if len(p.parameters2Slices(directive.GetParameters())) == 0 {
		return ""
	}

	return directive.GetParameters()[0].GetValue()
}

// GetLimitConn 获取并发连接数限制
func (p *Parser) GetLimitConn() [][]string {
	directives, err := p.Find("server.limit_conn")
	if err != nil {
		return nil
	}

	var result [][]string
	for _, dir := range directives {
		result = append(result, p.parameters2Slices(dir.GetParameters()))
	}

	return result
}

// GetBasicAuth 获取基本认证配置
func (p *Parser) GetBasicAuth() (string, string) {
	// auth_basic "realm"
	realmDir, err := p.FindOne("server.auth_basic")
	if err != nil {
		return "", ""
	}

	// auth_basic_user_file /path/to/file
	fileDir, err := p.FindOne("server.auth_basic_user_file")
	if err != nil {
		return "", ""
	}

	realm := ""
	if len(realmDir.GetParameters()) > 0 {
		realm = realmDir.GetParameters()[0].GetValue()
	}

	file := ""
	if len(fileDir.GetParameters()) > 0 {
		file = fileDir.GetParameters()[0].GetValue()
	}

	return realm, file
}
