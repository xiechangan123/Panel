package service

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/chix/v2"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/firewall"
	"github.com/acepanel/panel/v3/pkg/os"
)

type FirewallService struct {
	t        *gotext.Locale
	firewall firewall.Firewall
}

func NewFirewallService(i do.Injector) (*FirewallService, error) {
	return &FirewallService{
		t:        do.MustInvoke[*gotext.Locale](i),
		firewall: firewall.NewFirewall(),
	}, nil
}

func (s *FirewallService) GetStatus(w http.ResponseWriter, r *http.Request) {
	running, err := s.firewall.Status()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, running)
}

func (s *FirewallService) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.FirewallStatus](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if req.Status {
		err = s.firewall.Enable()
	} else {
		err = s.firewall.Disable()
	}

	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *FirewallService) GetRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.firewall.ListRule()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	var filledRules []map[string]any
	for rule := range slices.Values(rules) {
		// 去除IP规则
		if rule.PortStart == 1 && rule.PortEnd == 65535 {
			continue
		}
		isUse := false
		for port := rule.PortStart; port <= rule.PortEnd; port++ {
			switch rule.Protocol {
			case firewall.ProtocolTCP:
				isUse = os.TCPPortInUse(port)
			case firewall.ProtocolUDP:
				isUse = os.UDPPortInUse(port)
			default:
				isUse = os.TCPPortInUse(port) || os.UDPPortInUse(port)
			}
			if isUse {
				break
			}
		}
		filledRules = append(filledRules, map[string]any{
			"type":       rule.Type,
			"family":     rule.Family,
			"port_start": rule.PortStart,
			"port_end":   rule.PortEnd,
			"protocol":   rule.Protocol,
			"address":    rule.Address,
			"strategy":   rule.Strategy,
			"direction":  rule.Direction,
			"in_use":     isUse,
		})
	}

	paged, total := Paginate(r, filledRules)

	Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

func (s *FirewallService) CreateRule(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.FirewallRule](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.firewall.Port(firewall.FireInfo{
		Type: firewall.Type(req.Type), Family: req.Family, PortStart: req.PortStart, PortEnd: req.PortEnd, Protocol: firewall.Protocol(req.Protocol), Address: req.Address, Strategy: firewall.Strategy(req.Strategy), Direction: firewall.Direction(req.Direction),
	}, firewall.OperationAdd); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *FirewallService) DeleteRule(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.FirewallRule](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.firewall.Port(firewall.FireInfo{
		Type: firewall.Type(req.Type), Family: req.Family, PortStart: req.PortStart, PortEnd: req.PortEnd, Protocol: firewall.Protocol(req.Protocol), Address: req.Address, Strategy: firewall.Strategy(req.Strategy), Direction: firewall.Direction(req.Direction),
	}, firewall.OperationRemove); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

// ExportRules 导出端口规则为 xlsx
func (s *FirewallService) ExportRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.firewall.ListRule()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	_ = f.SetSheetRow(sheet, "A1", &[]any{"type", "family", "protocol", "port_start", "port_end", "address", "strategy", "direction"})
	row := 2
	for rule := range slices.Values(rules) {
		// 去除IP规则
		if rule.PortStart == 1 && rule.PortEnd == 65535 {
			continue
		}
		_ = f.SetSheetRow(sheet, fmt.Sprintf("A%d", row), &[]any{string(rule.Type), rule.Family, string(rule.Protocol), rule.PortStart, rule.PortEnd, rule.Address, string(rule.Strategy), string(rule.Direction)})
		row++
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="firewall_rules.xlsx"`)
	_ = f.Write(w)
}

// ImportRules 从 xlsx 导入端口规则
func (s *FirewallService) ImportRules(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	defer func() { _ = file.Close() }()

	f, err := excelize.OpenReader(file)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, s.t.Get("failed to parse xlsx: %v", err))
		return
	}
	defer func() { _ = f.Close() }()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	if len(rows) < 2 {
		Error(w, http.StatusUnprocessableEntity, s.t.Get("no rules found in file"))
		return
	}

	// 按表头定位列，兼容列顺序变化
	index := make(map[string]int)
	for i, name := range rows[0] {
		index[strings.TrimSpace(name)] = i
	}
	if _, ok := index["port_start"]; !ok {
		Error(w, http.StatusUnprocessableEntity, s.t.Get("invalid file: missing port_start column"))
		return
	}
	cell := func(row []string, name string) string {
		i, ok := index[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	succeeded, failed := 0, 0
	for _, row := range rows[1:] {
		portStart := cast.ToUint(cell(row, "port_start"))
		portEnd := cast.ToUint(cell(row, "port_end"))
		if portEnd == 0 {
			portEnd = portStart
		}
		if portStart < 1 || portEnd > 65535 || portStart > portEnd {
			failed++
			continue
		}
		info := firewall.FireInfo{
			Type:      firewall.Type(cmp.Or(cell(row, "type"), string(firewall.TypeNormal))),
			Family:    cmp.Or(cell(row, "family"), "ipv4"),
			Address:   cell(row, "address"),
			PortStart: portStart,
			PortEnd:   portEnd,
			Protocol:  firewall.Protocol(cmp.Or(cell(row, "protocol"), string(firewall.ProtocolTCP))),
			Strategy:  firewall.Strategy(cmp.Or(cell(row, "strategy"), string(firewall.StrategyAccept))),
			Direction: firewall.Direction(cmp.Or(cell(row, "direction"), string(firewall.DirectionIn))),
		}
		if err = s.firewall.Port(info, firewall.OperationAdd); err != nil {
			failed++
			continue
		}
		succeeded++
	}

	Success(w, chix.M{
		"succeeded": succeeded,
		"failed":    failed,
	})
}

func (s *FirewallService) GetIPRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.firewall.ListRule()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	filledRules := lo.FilterMap(rules, func(rule firewall.FireInfo, _ int) (map[string]any, bool) {
		// 保留IP规则
		if rule.PortStart != 1 || rule.PortEnd != 65535 || rule.Address == "" {
			return nil, false
		}
		return map[string]any{
			"family":    rule.Family,
			"protocol":  rule.Protocol,
			"address":   rule.Address,
			"strategy":  rule.Strategy,
			"direction": rule.Direction,
		}, true
	})

	paged, total := Paginate(r, filledRules)

	Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

func (s *FirewallService) CreateIPRule(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.FirewallIPRule](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.firewall.RichRules(firewall.FireInfo{
		Family: req.Family, Address: req.Address, Protocol: firewall.Protocol(req.Protocol), Strategy: firewall.Strategy(req.Strategy), Direction: firewall.Direction(req.Direction),
	}, firewall.OperationAdd); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *FirewallService) DeleteIPRule(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.FirewallIPRule](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.firewall.RichRules(firewall.FireInfo{
		Family: req.Family, Address: req.Address, Protocol: firewall.Protocol(req.Protocol), Strategy: firewall.Strategy(req.Strategy), Direction: firewall.Direction(req.Direction),
	}, firewall.OperationRemove); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *FirewallService) GetForwards(w http.ResponseWriter, r *http.Request) {
	forwards, err := s.firewall.ListForward()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	paged, total := Paginate(r, forwards)

	Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

func (s *FirewallService) CreateForward(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.FirewallForward](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.firewall.Forward(firewall.Forward{
		Protocol: firewall.Protocol(req.Protocol), Port: req.Port, TargetIP: req.TargetIP, TargetPort: req.TargetPort,
	}, firewall.OperationAdd); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *FirewallService) DeleteForward(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.FirewallForward](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.firewall.Forward(firewall.Forward{
		Protocol: firewall.Protocol(req.Protocol), Port: req.Port, TargetIP: req.TargetIP, TargetPort: req.TargetPort,
	}, firewall.OperationRemove); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *FirewallService) GetPortUsage(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.ParseUint(r.URL.Query().Get("port"), 10, 32)
	if err != nil || port == 0 || port > 65535 {
		Error(w, http.StatusUnprocessableEntity, "invalid port")
		return
	}

	protocol := r.URL.Query().Get("protocol")
	if protocol != "tcp" && protocol != "udp" {
		protocol = "tcp"
	}

	Success(w, os.GetPortProcess(uint(port), protocol))
}
