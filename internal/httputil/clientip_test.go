package httputil

import (
	"net/http"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		realIP     string
		xff        string
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Real-IP 优先",
			realIP:     "1.2.3.4",
			xff:        "9.9.9.9",
			remoteAddr: "172.18.0.3:56236",
			want:       "1.2.3.4",
		},
		{
			name:       "X-Forwarded-For 多段取第一个",
			xff:        "1.2.3.4, 10.0.0.1, 172.18.0.3",
			remoteAddr: "172.18.0.3:56236",
			want:       "1.2.3.4",
		},
		{
			name:       "X-Forwarded-For 单段",
			xff:        "1.2.3.4",
			remoteAddr: "172.18.0.3:56236",
			want:       "1.2.3.4",
		},
		{
			name:       "无头回退 RemoteAddr(带端口)",
			remoteAddr: "1.2.3.4:56236",
			want:       "1.2.3.4",
		},
		{
			name:       "无头回退 RemoteAddr(不带端口)",
			remoteAddr: "1.2.3.4",
			want:       "1.2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.realIP != "" {
				r.Header.Set("X-Real-IP", tt.realIP)
			}
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := ClientIP(r); got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
