package goatcounter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(GoatCounter{})
	httpcaddyfile.RegisterHandlerDirective("goatcounter", parseCaddyfile)
}

type GoatCounter struct {
	APIHost   string   `json:"api_host,omitempty"`
	Site      string   `json:"site,omitempty"`
	Token     string   `json:"token,omitempty"`
	Paths     []string `json:"paths,omitempty"`
	UserAgent string   `json:"user_agent,omitempty"`
	logger    *zap.Logger
	client    *http.Client
}

type Hit struct {
	Path      string `json:"path"`
	Query     string `json:"query,omitempty"`
	Ref       string `json:"ref,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	IP        string `json:"ip,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type CountRequest struct {
	NoSessions bool  `json:"no_sessions"`
	Hits       []Hit `json:"hits"`
}

func (GoatCounter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.goatcounter",
		New: func() caddy.Module { return new(GoatCounter) },
	}
}

func (gc *GoatCounter) Provision(ctx caddy.Context) error {
	gc.logger = ctx.Logger(gc)
	gc.client = &http.Client{
		Timeout: 5 * time.Second,
	}

	if gc.UserAgent == "" {
		gc.UserAgent = "caddy-goatcounter/1.0"
	}

	if gc.APIHost == "" {
		return fmt.Errorf("goatcounter api_host is required")
	}

	if gc.Site == "" {
		return fmt.Errorf("goatcounter site is required")
	}

	// Ensure APIHost has protocol
	if !strings.Contains(gc.APIHost, "://") {
		gc.APIHost = "http://" + gc.APIHost
	}

	gc.logger.Info("GoatCounter handler provisioned",
		zap.String("api_host", gc.APIHost),
		zap.String("site", gc.Site),
		zap.Strings("paths", gc.Paths))

	return nil
}

func (gc *GoatCounter) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	shouldTrack := false

	if len(gc.Paths) == 0 {
		shouldTrack = true
	} else {
		for _, path := range gc.Paths {
			if strings.HasPrefix(r.URL.Path, path) {
				shouldTrack = true
				break
			}
		}
	}

	if shouldTrack {
		go gc.trackRequest(r)
	}

	return next.ServeHTTP(w, r)
}

func (gc *GoatCounter) trackRequest(r *http.Request) {
	// Extract IP address
	ip := gc.getClientIP(r)

	// Create hit data
	hit := Hit{
		Path:      r.URL.Path,
		UserAgent: r.Header.Get("User-Agent"),
		IP:        ip,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	if r.URL.RawQuery != "" {
		hit.Query = r.URL.RawQuery
	}

	if referer := r.Header.Get("Referer"); referer != "" {
		hit.Ref = referer
	}

	// Create API request
	countReq := CountRequest{
		NoSessions: true,
		Hits:       []Hit{hit},
	}

	jsonData, err := json.Marshal(countReq)
	if err != nil {
		gc.logger.Error("failed to marshal tracking request", zap.Error(err))
		return
	}

	apiURL := fmt.Sprintf("%s/api/v0/count", strings.TrimSuffix(gc.APIHost, "/"))
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		gc.logger.Error("failed to create tracking request", zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", gc.UserAgent)
	req.Header.Set("Host", gc.Site)

	if gc.Token != "" {
		req.Header.Set("Authorization", "Bearer "+gc.Token)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := gc.client.Do(req)
	if err != nil {
		gc.logger.Debug("failed to send tracking request", zap.Error(err))
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			gc.logger.Debug("failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode >= 400 {
		gc.logger.Debug("tracking request failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("path", r.URL.Path))
	} else {
		gc.logger.Debug("tracking request successful",
			zap.String("path", r.URL.Path))
	}
}

func (gc *GoatCounter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, get the first one
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

func (gc *GoatCounter) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "api_host":
				if !d.NextArg() {
					return d.ArgErr()
				}
				gc.APIHost = d.Val()
			case "site":
				if !d.NextArg() {
					return d.ArgErr()
				}
				gc.Site = d.Val()
			case "token":
				if !d.NextArg() {
					return d.ArgErr()
				}
				gc.Token = d.Val()
			case "paths":
				gc.Paths = d.RemainingArgs()
			case "user_agent":
				if !d.NextArg() {
					return d.ArgErr()
				}
				gc.UserAgent = d.Val()
			default:
				return d.Errf("unknown directive: %s", d.Val())
			}
		}
	}

	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var gc GoatCounter
	err := gc.UnmarshalCaddyfile(h.Dispenser)
	return &gc, err
}

var (
	_ caddy.Provisioner           = (*GoatCounter)(nil)
	_ caddyhttp.MiddlewareHandler = (*GoatCounter)(nil)
	_ caddyfile.Unmarshaler       = (*GoatCounter)(nil)
)