package web

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"

	"vps-monitor/internal/cluster"
)

func (s *Server) requireInternalRequest(r *http.Request) error {
	// When an internal token is configured, always require it.
	// This is critical when running behind a reverse proxy (e.g. Nginx)
	// where RemoteAddr is always 127.0.0.1.
	if token := s.cfg.InternalToken(); token != "" {
		auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(auth), []byte(token)) != 1 {
			return fmt.Errorf("invalid or missing internal token")
		}
		return nil
	}

	// Fallback: IP-based access control (only safe without reverse proxy)
	host := remoteHost(r.RemoteAddr)
	if host == "" {
		return fmt.Errorf("unable to determine remote host")
	}
	if isLoopbackHost(host) {
		return nil
	}

	allowed := map[string]struct{}{}
	members, err := s.cluster.ActiveMembers(r.Context())
	if err == nil {
		for _, member := range members {
			if member.PublicIPv4 != "" {
				allowed[member.PublicIPv4] = struct{}{}
			}
			if apiHost := remoteHost(member.APIAddr); apiHost != "" {
				allowed[apiHost] = struct{}{}
			}
		}
	}
	for _, peer := range s.cfg.Cluster.Peers {
		if peer.PublicIPv4 != "" {
			allowed[peer.PublicIPv4] = struct{}{}
		}
		if apiHost := remoteHost(peer.APIAddr); apiHost != "" {
			allowed[apiHost] = struct{}{}
		}
	}
	if _, ok := allowed[host]; ok {
		return nil
	}
	return fmt.Errorf("internal endpoint forbidden for remote host %s", host)
}

func (s *Server) isAllowedInternalCommand(cmdType string) bool {
	switch cmdType {
	case cluster.CommandAdminSettings,
		cluster.CommandAdminSession,
		cluster.CommandDeleteSession,
		cluster.CommandMonitorCheck,
		cluster.CommandDeleteCheck,
		cluster.CommandNodeDisplayName,
		cluster.CommandDeleteNodeName:
		return true
	default:
		return false
	}
}

func remoteHost(value string) string {
	if value == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		return strings.TrimSpace(host)
	}
	return strings.TrimSpace(value)
}

func isLoopbackHost(host string) bool {
	switch strings.TrimSpace(host) {
	case "127.0.0.1", "::1", "[::1]", "localhost":
		return true
	default:
		return false
	}
}
