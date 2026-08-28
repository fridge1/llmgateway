package httputil

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP 提取请求的真实客户端 IP。本服务只经由自有 nginx 暴露(backend 端口不对外),
// nginx 通过 X-Real-IP / X-Forwarded-For 转发真实 IP,因此优先信任这两个头,
// 回退到 r.RemoteAddr(直连场景或头缺失时)。
//
// 注意:若未来 backend 直连公网,客户端可伪造这两个头,需重新评估信任策略。
func ClientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, found := strings.Cut(xff, ","); found {
			if ip := strings.TrimSpace(first); ip != "" {
				return ip
			}
		} else if ip := strings.TrimSpace(xff); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
