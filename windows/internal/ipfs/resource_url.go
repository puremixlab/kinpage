package ipfs

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// applyWebUICORS ensures the local Kubo RPC API allows https://webui.ipfs.io
// so users can inspect their local node in IPFS WebUI, matching macOS Planet.
func applyWebUICORS(config map[string]json.RawMessage) error {
	var api map[string]json.RawMessage
	if len(config["API"]) > 0 && string(bytes.TrimSpace(config["API"])) != "null" {
		if json.Unmarshal(config["API"], &api) != nil {
			return errors.New("Kubo API 配置无效")
		}
	}
	if api == nil {
		api = map[string]json.RawMessage{}
	}
	var headers map[string]json.RawMessage
	if len(api["HTTPHeaders"]) > 0 && string(bytes.TrimSpace(api["HTTPHeaders"])) != "null" {
		if json.Unmarshal(api["HTTPHeaders"], &headers) != nil {
			return errors.New("Kubo API.HTTPHeaders 配置无效")
		}
	}
	if headers == nil {
		headers = map[string]json.RawMessage{}
	}
	var origins []string
	if len(headers["Access-Control-Allow-Origin"]) > 0 {
		_ = json.Unmarshal(headers["Access-Control-Allow-Origin"], &origins)
	}
	hasWebUI := false
	for _, o := range origins {
		if strings.EqualFold(strings.TrimSpace(o), "https://webui.ipfs.io") {
			hasWebUI = true
			break
		}
	}
	if !hasWebUI {
		origins = append(origins, "https://webui.ipfs.io")
		raw, err := json.Marshal(origins)
		if err != nil {
			return err
		}
		headers["Access-Control-Allow-Origin"] = raw
	}
	if _, exists := headers["Access-Control-Allow-Methods"]; !exists {
		headers["Access-Control-Allow-Methods"] = json.RawMessage(`["PUT","POST","GET"]`)
	}
	rawHeaders, err := json.Marshal(headers)
	if err != nil {
		return err
	}
	api["HTTPHeaders"] = rawHeaders
	rawAPI, err := json.Marshal(api)
	if err != nil {
		return err
	}
	config["API"] = rawAPI
	return nil
}

// ResolveResourceURL normalizes a CID, IPNS name, ENS/SNS/.bit/.fc domain, or
// ipfs:// / ipns:// URI into a safe local gateway URL.
func ResolveResourceURL(gateway, raw string) (string, error) {
	gw := strings.TrimRight(strings.TrimSpace(gateway), "/")
	if gw == "" {
		return "", errors.New("本地 IPFS 网关尚未就绪，请先启动 IPFS 节点")
	}
	u, err := url.Parse(gw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("本地 IPFS 网关地址无效")
	}

	input := strings.TrimSpace(raw)
	if input == "" {
		return "", errors.New("请输入 CID、IPNS 或去中心化域名")
	}
	if strings.ContainsAny(input, "\r\n\t \"'<>\\") {
		return "", errors.New("IPFS 资源地址包含无效字符")
	}

	lower := strings.ToLower(input)
	switch {
	case strings.HasPrefix(lower, "ipfs://"):
		rest := strings.TrimLeft(input[len("ipfs://"):], "/")
		if rest == "" {
			return "", errors.New("CID 不能为空")
		}
		return gw + "/ipfs/" + rest, nil
	case strings.HasPrefix(lower, "ipns://"):
		rest := strings.TrimLeft(input[len("ipns://"):], "/")
		if rest == "" {
			return "", errors.New("IPNS 地址不能为空")
		}
		return gw + "/ipns/" + rest, nil
	case strings.HasPrefix(lower, "/ipfs/") || strings.HasPrefix(lower, "/ipns/"):
		return gw + input, nil
	case strings.Contains(input, "://"):
		return "", errors.New("仅支持 CID、IPNS、ipfs://、ipns:// 或 .eth/.sol/.bit/.fc 域名")
	}

	firstSeg := input
	if idx := strings.IndexByte(firstSeg, '/'); idx >= 0 {
		firstSeg = firstSeg[:idx]
	}
	if firstSeg == "" || firstSeg == "." || firstSeg == ".." || strings.Contains(firstSeg, ":") {
		return "", errors.New("IPFS 资源地址无效")
	}

	// CIDv0 (Qm...) or CIDv1 (bafy..., bafk..., bafz...) -> /ipfs/, otherwise /ipns/.
	firstLower := strings.ToLower(firstSeg)
	if strings.HasPrefix(firstSeg, "Qm") && len(firstSeg) >= 46 {
		return gw + "/ipfs/" + input, nil
	}
	if strings.HasPrefix(firstLower, "bafy") || strings.HasPrefix(firstLower, "bafk") || strings.HasPrefix(firstLower, "bafz") || strings.HasPrefix(firstLower, "bafb") {
		return gw + "/ipfs/" + input, nil
	}
	return gw + "/ipns/" + input, nil
}
