package node

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const (
	defaultScriptTimeout = 300 * time.Millisecond
	maxScriptProxies     = 2000
	maxScriptFieldLen    = 512
)

// RunProxyScript executes a user supplied operator(proxies) script in a
// sandboxed goja runtime and returns the transformed proxy list.
//
// The runtime only exposes a plain array of objects (name/type/server/port/
// udp/tfo/skipCertVerify/link plus a hidden _id). It has no access to require,
// the filesystem, the network or any host object, and is interrupted after the
// timeout. Nodes the script returns without a valid _id are only kept if they
// carry a full share link.
func RunProxyScript(code string, proxies []Outbound, timeout time.Duration) ([]Outbound, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return proxies, nil
	}
	if timeout <= 0 {
		timeout = defaultScriptTimeout
	}

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	arr := make([]map[string]interface{}, 0, len(proxies))
	byID := make(map[int]Outbound, len(proxies))
	for i, p := range proxies {
		id := p.ScriptID
		if id <= 0 {
			id = i + 1
		}
		arr = append(arr, map[string]interface{}{
			"name":           p.Name,
			"type":           p.Type,
			"server":         p.Server,
			"port":           p.Port,
			"udp":            p.UDP,
			"tfo":            p.TFO,
			"skipCertVerify": p.SkipCertVerify,
			"link":           p.Link,
			"_id":            id,
		})
		byID[id] = p
	}
	_ = vm.Set("proxies", arr)

	if _, err := vm.RunString(code); err != nil {
		return nil, fmt.Errorf("脚本执行失败: %w", err)
	}
	opVal := vm.Get("operator")
	if opVal == nil {
		return nil, fmt.Errorf("脚本必须定义 operator(proxies) 函数")
	}
	op, ok := goja.AssertFunction(opVal)
	if !ok {
		return nil, fmt.Errorf("operator 必须是函数")
	}

	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt("脚本执行超时")
	})
	defer timer.Stop()

	result, err := op(goja.Undefined(), vm.Get("proxies"))
	if err != nil {
		return nil, fmt.Errorf("脚本执行失败: %w", err)
	}
	exported, err := exportProxyArray(result.Export())
	if err != nil {
		return nil, err
	}
	if len(exported) > maxScriptProxies {
		return nil, fmt.Errorf("脚本返回节点数超过上限 %d", maxScriptProxies)
	}

	out := make([]Outbound, 0, len(exported))
	for _, item := range exported {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		p, err := proxyFromScript(obj, byID)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// exportProxyArray normalizes goja export results. A JS array built by map/
// filter exports as []interface{}, while a wrapped Go slice passed into the VM
// and returned unchanged exports as its original Go type.
func exportProxyArray(v interface{}) ([]interface{}, error) {
	switch arr := v.(type) {
	case []interface{}:
		return arr, nil
	case []map[string]interface{}:
		out := make([]interface{}, 0, len(arr))
		for _, m := range arr {
			out = append(out, m)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("operator 必须返回节点数组")
	}
}

func proxyFromScript(obj map[string]interface{}, byID map[int]Outbound) (Outbound, error) {
	base, hasBase := Outbound{}, false
	if id, ok := scriptInt(obj["_id"]); ok && id > 0 {
		if p, found := byID[id]; found {
			base, hasBase = p, true
			base.ScriptID = id
		}
	}
	if !hasBase {
		base.ScriptID = 0
		link := truncate(scriptString(obj["link"]), 4096)
		if link == "" || !strings.Contains(link, "://") {
			return Outbound{}, fmt.Errorf("脚本新增节点缺少有效 link")
		}
		p, err := ParseOutbound(link)
		if err != nil {
			return Outbound{}, err
		}
		base = p
	}
	if name := truncate(scriptString(obj["name"]), maxScriptFieldLen); name != "" {
		base.Name = name
	}
	if server := truncate(scriptString(obj["server"]), maxScriptFieldLen); server != "" {
		base.Server = server
	}
	if port, ok := scriptInt(obj["port"]); ok && port > 0 {
		base.Port = port
	}
	if v, ok := obj["udp"].(bool); ok {
		base.UDP = v
	}
	if v, ok := obj["tfo"].(bool); ok {
		base.TFO = v
	}
	if v, ok := obj["skipCertVerify"].(bool); ok {
		base.SkipCertVerify = v
	}
	if base.Name == "" {
		base.Name = fmt.Sprintf("%s:%d", base.Server, base.Port)
	}
	base.Link = base.ToLink()
	return base, nil
}

func scriptString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func scriptInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max]
	}
	return s
}
