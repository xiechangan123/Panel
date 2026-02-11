package firewall

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/spf13/cast"

	"github.com/acepanel/panel/pkg/shell"
	"github.com/acepanel/panel/pkg/systemctl"
)

type firewalld struct {
	forwardListRegex *regexp.Regexp
	richRuleRegex    *regexp.Regexp
}

func newFirewalld() *firewalld {
	return &firewalld{
		forwardListRegex: regexp.MustCompile(`^port=(\d{1,5}):proto=(.+?):toport=(\d{1,5}):toaddr=(.*)$`),
		richRuleRegex:    regexp.MustCompile(`^rule family="([^"]+)"(?: .*?(source|destination) address="([^"]+)")?(?: .*?port port="([^"]+)")?(?: .*?protocol(?: value)?="([^"]+)")?.*?(accept|drop|reject|mark).*?$`),
	}
}

func (r *firewalld) Status() (bool, error) {
	return systemctl.Status("firewalld")
}

func (r *firewalld) Enable() error {
	if err := systemctl.Start("firewalld"); err != nil {
		return err
	}
	return systemctl.Enable("firewalld")
}

func (r *firewalld) Disable() error {
	if err := systemctl.Stop("firewalld"); err != nil {
		return err
	}
	return systemctl.Disable("firewalld")
}

func (r *firewalld) ListRule() ([]FireInfo, error) {
	var wg sync.WaitGroup
	var portRules []FireInfo
	var richRules []FireInfo
	wg.Add(2)

	go func() {
		defer wg.Done()
		out, err := shell.Execf("firewall-cmd --zone=public --list-ports")
		if err != nil {
			return
		}
		ports := strings.Split(out, " ")
		for _, port := range ports {
			if len(port) == 0 {
				continue
			}
			var item FireInfo
			item.Type = TypeNormal
			if strings.Contains(port, "/") {
				ruleItem := strings.Split(port, "/")
				portItem := strings.Split(ruleItem[0], "-")
				if len(portItem) > 1 {
					item.PortStart = cast.ToUint(portItem[0])
					item.PortEnd = cast.ToUint(portItem[1])
				} else {
					item.PortStart = cast.ToUint(ruleItem[0])
					item.PortEnd = cast.ToUint(ruleItem[0])
				}
				item.Protocol = Protocol(ruleItem[1])
			}
			item.Family = "ipv4"
			item.Strategy = "accept"
			item.Direction = "in"
			portRules = append(portRules, item)
		}
	}()
	go func() {
		defer wg.Done()
		rich, err := r.listRichRule()
		if err != nil {
			return
		}
		richRules = rich
	}()

	wg.Wait()

	data := make([]FireInfo, 0, len(portRules)+len(richRules))
	data = append(data, portRules...)
	data = append(data, richRules...)

	slices.SortFunc(data, func(a FireInfo, b FireInfo) int {
		if a.PortStart != b.PortStart {
			return cmp.Compare(a.PortStart, b.PortStart)
		}
		if a.PortEnd != b.PortEnd {
			return cmp.Compare(a.PortEnd, b.PortEnd)
		}
		if a.Protocol != b.Protocol {
			return strings.Compare(string(a.Protocol), string(b.Protocol))
		}
		if a.Family != b.Family {
			return strings.Compare(a.Family, b.Family)
		}
		if a.Strategy != b.Strategy {
			return strings.Compare(string(a.Strategy), string(b.Strategy))
		}
		if a.Direction != b.Direction {
			return strings.Compare(string(a.Direction), string(b.Direction))
		}
		if a.Type != b.Type {
			return strings.Compare(string(a.Type), string(b.Type))
		}
		return 0
	})

	return mergeRules(data), nil
}

func (r *firewalld) ListForward() ([]FireForwardInfo, error) {
	out, err := shell.Execf("firewall-cmd --zone=public --list-forward-ports")
	if err != nil {
		return nil, err
	}

	var data []FireForwardInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimFunc(line, func(r rune) bool {
			return r <= 32
		})
		if r.forwardListRegex.MatchString(line) {
			match := r.forwardListRegex.FindStringSubmatch(line)
			if len(match) < 4 {
				continue
			}
			if len(match[4]) == 0 {
				match[4] = "127.0.0.1"
			}
			data = append(data, FireForwardInfo{
				Port:       cast.ToUint(match[1]),
				Protocol:   Protocol(match[2]),
				TargetIP:   match[4],
				TargetPort: cast.ToUint(match[3]),
			})
		}
	}

	slices.SortFunc(data, func(a FireForwardInfo, b FireForwardInfo) int {
		if a.Port != b.Port {
			return cmp.Compare(a.Port, b.Port)
		}
		if a.TargetPort != b.TargetPort {
			return cmp.Compare(a.TargetPort, b.TargetPort)
		}
		if a.Protocol != b.Protocol {
			return strings.Compare(string(a.Protocol), string(b.Protocol))
		}
		if a.TargetIP != b.TargetIP {
			return strings.Compare(a.TargetIP, b.TargetIP)
		}
		return 0
	})

	return data, nil
}

func (r *firewalld) listRichRule() ([]FireInfo, error) {
	out, err := shell.Execf("firewall-cmd --zone=public --list-rich-rules")
	if err != nil {
		return nil, err
	}

	var data []FireInfo
	rules := strings.Split(out, "\n")
	for _, rule := range rules {
		if len(rule) == 0 {
			continue
		}
		if richRules, err := r.parseRichRule(rule); err == nil {
			data = append(data, richRules)
		}
	}

	return data, nil
}

func (r *firewalld) Port(rule FireInfo, operation Operation) error {
	if rule.PortEnd == 0 {
		rule.PortEnd = rule.PortStart
	}
	if rule.PortStart > rule.PortEnd {
		return fmt.Errorf("invalid port range: %d-%d", rule.PortStart, rule.PortEnd)
	}
	// 不支持的切换使用rich rules
	if (rule.Family != "" && rule.Family != "ipv4") || rule.Direction != "in" || rule.Address != "" || rule.Strategy != "accept" || rule.Type == TypeRich {
		return r.RichRules(rule, operation)
	}

	// 未设置协议默认为tcp/udp
	if rule.Protocol == "" {
		rule.Protocol = ProtocolTCPUDP
	}

	// 删除时忽略错误（规则可能已不存在）
	for _, protocol := range buildProtocols(rule.Protocol) {
		_, err := shell.Execf("firewall-cmd --zone=public --%s-port=%d-%d/%s --permanent", operation, rule.PortStart, rule.PortEnd, protocol)
		if err != nil && operation != OperationRemove {
			return err
		}
	}

	_, err := shell.Execf("firewall-cmd --reload")
	return err
}

func (r *firewalld) RichRules(rule FireInfo, operation Operation) error {
	// 出站规则下，必须指定具体的地址，否则会添加成入站规则
	if rule.Direction == "out" && rule.Address == "" {
		return fmt.Errorf("outbound rules must specify an address")
	}

	for _, protocol := range buildProtocols(rule.Protocol) {
		cmd := r.buildRichRuleStr(rule, protocol)
		_, err := shell.Execf("firewall-cmd --zone=public --%s-rich-rule '%s' --permanent", operation, cmd)
		if err != nil && operation != OperationRemove {
			return err
		}
	}

	_, err := shell.Execf("firewall-cmd --reload")
	return err
}

// buildRichRuleStr 构建 firewalld 富规则字符串
func (r *firewalld) buildRichRuleStr(rule FireInfo, protocol string) string {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, `rule family="%s" `, rule.Family)

	if rule.Address != "" {
		switch rule.Direction {
		case "out":
			_, _ = fmt.Fprintf(&sb, `destination address="%s" `, rule.Address)
		default:
			_, _ = fmt.Fprintf(&sb, `source address="%s" `, rule.Address)
		}
	}
	if rule.PortStart != 0 && rule.PortEnd != 0 && (rule.PortStart != 1 || rule.PortEnd != 65535) {
		_, _ = fmt.Fprintf(&sb, `port port="%d-%d" `, rule.PortStart, rule.PortEnd)
	}
	if protocol != "" {
		sb.WriteString(`protocol`)
		if rule.PortStart == 0 && rule.PortEnd == 0 { // IP 规则下，必须添加 value
			sb.WriteString(` value`)
		}
		_, _ = fmt.Fprintf(&sb, `="%s" `, protocol)
	}

	sb.WriteString(string(rule.Strategy))
	return sb.String()
}

func (r *firewalld) Forward(rule Forward, operation Operation) error {
	if err := r.enableForward(); err != nil {
		return err
	}

	// 启用 IP 转发
	_, _ = shell.Execf("sysctl -w net.ipv4.ip_forward=1")
	_, _ = shell.Execf("sysctl -w net.ipv6.conf.all.forwarding=1")
	_ = os.WriteFile("/etc/sysctl.d/99-acepanel-forward.conf", []byte("net.ipv4.ip_forward=1\nnet.ipv6.conf.all.forwarding=1\n"), 0644)

	for _, protocol := range buildProtocols(rule.Protocol) {
		var cmd string
		if rule.TargetIP != "" && !isLocalAddress(rule.TargetIP) {
			cmd = fmt.Sprintf("firewall-cmd --zone=public --%s-forward-port=port=%d:proto=%s:toport=%d:toaddr=%s --permanent", operation, rule.Port, protocol, rule.TargetPort, rule.TargetIP)
		} else {
			cmd = fmt.Sprintf("firewall-cmd --zone=public --%s-forward-port=port=%d:proto=%s:toport=%d --permanent", operation, rule.Port, protocol, rule.TargetPort)
		}
		_, err := shell.Exec(cmd)
		if err != nil && operation != OperationRemove {
			return err
		}
	}

	_, err := shell.Execf("firewall-cmd --reload")
	return err
}

func (r *firewalld) PingStatus() (bool, error) {
	out, err := shell.Execf("firewall-cmd --zone=public --list-rich-rules")
	if err != nil { // 可能防火墙已关闭等
		return true, nil
	}

	if !strings.Contains(out, `rule protocol value="icmp" drop`) {
		return true, nil
	}

	return false, nil
}

func (r *firewalld) UpdatePingStatus(status bool) error {
	var err error
	if status {
		_, err = shell.Execf(`firewall-cmd --zone=public --permanent --remove-rich-rule='rule protocol value=icmp drop'`)
	} else {
		_, err = shell.Execf(`firewall-cmd --zone=public --permanent --add-rich-rule='rule protocol value=icmp drop'`)
	}
	if err != nil {
		return err
	}

	_, err = shell.Execf("firewall-cmd --reload")
	return err
}

func (r *firewalld) parseRichRule(line string) (FireInfo, error) {
	if !r.richRuleRegex.MatchString(line) {
		return FireInfo{}, errors.New("invalid rich rule format")
	}

	match := r.richRuleRegex.FindStringSubmatch(line)
	if len(match) < 7 {
		return FireInfo{}, errors.New("invalid rich rule")
	}

	fireInfo := FireInfo{
		Type:     TypeRich,
		Family:   match[1],
		Address:  match[3],
		Protocol: Protocol(match[5]),
		Strategy: Strategy(match[6]),
	}

	if match[2] == "destination" {
		fireInfo.Direction = "out"
	} else {
		fireInfo.Direction = "in"
	}
	if fireInfo.Protocol == "" {
		fireInfo.Protocol = "tcp/udp"
	}

	ports := strings.Split(match[4], "-")
	if len(ports) == 2 { // 添加端口范围
		fireInfo.PortStart = cast.ToUint(ports[0])
		fireInfo.PortEnd = cast.ToUint(ports[1])
	} else if len(ports) == 1 && ports[0] != "" { // 添加单个端口
		port := cast.ToUint(ports[0])
		fireInfo.PortStart = port
		fireInfo.PortEnd = port
	} else if len(ports) == 1 && ports[0] == "" { // 未添加端口规则，表示所有端口
		fireInfo.PortStart = 1
		fireInfo.PortEnd = 65535
	}

	return fireInfo, nil
}

func (r *firewalld) enableForward() error {
	out, err := shell.Execf("firewall-cmd --zone=public --query-masquerade")
	if err != nil {
		if out == "no" {
			out, err = shell.Execf("firewall-cmd --zone=public --add-masquerade --permanent")
			if err != nil {
				return fmt.Errorf("%v: %s", err, out)
			}
			_, err = shell.Execf("firewall-cmd --reload")
			return err
		}
		return fmt.Errorf("%v: %s", err, out)
	}

	return nil
}
