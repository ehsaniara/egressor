package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ehsaniara/egressor/internal/audit"
	"github.com/ehsaniara/egressor/internal/ca"
	"github.com/ehsaniara/egressor/internal/extract"
	"github.com/ehsaniara/egressor/internal/policy"
)

// Interceptor performs TLS interception (MITM) to capture HTTP traffic.
type Interceptor struct {
	certCache        *ca.CertCache
	logBody          bool
	maxBody          int
	policy           *policy.Engine
	resolver         policy.PromptResolver
	skipContentTypes []string
}

// NewInterceptor creates an interceptor backed by the given CA authority.
func NewInterceptor(authority *ca.Authority, logBody bool, maxBody int, pol *policy.Engine, skipContentTypes []string) *Interceptor {
	return &Interceptor{
		certCache:        ca.NewCertCache(authority),
		logBody:          logBody,
		maxBody:          maxBody,
		policy:           pol,
		skipContentTypes: skipContentTypes,
	}
}

// SetResolver sets the prompt resolver for interactive content keyword approval.
func (i *Interceptor) SetResolver(r policy.PromptResolver) {
	i.resolver = r
}

// Intercept performs TLS MITM on an established CONNECT tunnel.
// clientConn has already received "200 Connection Established".
// upstreamConn is a raw TCP connection to the target server.
func (i *Interceptor) Intercept(clientConn net.Conn, upstreamConn net.Conn, host string, sess *audit.Session) error {
	// TLS-terminate the client side with a dynamic certificate
	clientTLS := tls.Server(clientConn, &tls.Config{
		GetCertificate: i.certCache.GetCertificate,
		NextProtos:     []string{"http/1.1"},
	})
	clientConn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := clientTLS.Handshake(); err != nil {
		return fmt.Errorf("client TLS handshake: %w", err)
	}
	clientConn.SetDeadline(time.Time{}) // clear deadline

	// TLS-connect to the real upstream server
	upstreamTLS := tls.Client(upstreamConn, &tls.Config{
		ServerName: host,
		NextProtos: []string{"http/1.1"},
	})
	if err := upstreamTLS.Handshake(); err != nil {
		return fmt.Errorf("upstream TLS handshake: %w", err)
	}

	defer clientTLS.Close()
	defer upstreamTLS.Close()

	// HTTP/1.1 relay loop
	clientReader := bufio.NewReader(clientTLS)

	for {
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading request: %w", err)
		}

		exchange := audit.InterceptedExchange{
			Timestamp:      time.Now(),
			Method:         req.Method,
			URL:            fmt.Sprintf("https://%s%s", host, req.URL.RequestURI()),
			RequestHeaders: flattenHeaders(req.Header),
		}

		// Read full request body into buffer before forwarding
		var reqBodyBuf bytes.Buffer
		if req.Body != nil {
			io.Copy(&limitWriter{w: &reqBodyBuf, max: i.maxBody}, req.Body)
			req.Body.Close()
		}
		bodyStr := reqBodyBuf.String()

		// Skip content scanning for binary/non-text content types
		skipScan := i.shouldSkipContentScan(req.Header.Get("Content-Type"))

		// Extract file references from the request payload
		files := extract.FilesFromBody(bodyStr)
		if len(files) > 0 && !skipScan {
			for _, f := range files {
				exchange.DetectedFiles = append(exchange.DetectedFiles, audit.FileRef{
					Path:   f.Path,
					Source: f.Source,
				})
			}
			slog.Info("files detected in payload",
				"session", sess.ID,
				"url", exchange.URL,
				"files", len(files),
			)

			paths := make([]string, len(files))
			for idx, f := range files {
				paths[idx] = f.Path
			}

			// Check directory scope — block if any file is outside allowed directories
			decision := i.policy.EvaluateScope(paths)
			if !decision.Allowed {
				slog.Warn("request blocked by directory scope policy",
					"session", sess.ID,
					"url", exchange.URL,
					"reason", decision.Reason,
				)
				exchange.StatusCode = 403
				exchange.Blocked = true
				exchange.BlockReason = decision.Reason
				if i.logBody {
					exchange.RequestBody = truncateBody(bodyStr, i.maxBody)
				}
				resp403 := &http.Response{
					StatusCode: 403,
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header:     http.Header{"Content-Type": {"text/plain"}},
					Body:       io.NopCloser(strings.NewReader("blocked by egressor: " + decision.Reason)),
				}
				resp403.Write(clientTLS)
				sess.Exchanges = append(sess.Exchanges, exchange)
				return nil
			}

			// Check file deny patterns — block if matched
			decision = i.policy.EvaluateFiles(paths)
			if !decision.Allowed {
				slog.Warn("request blocked by file policy",
					"session", sess.ID,
					"url", exchange.URL,
					"reason", decision.Reason,
				)
				exchange.StatusCode = 403
				exchange.Blocked = true
				exchange.BlockReason = decision.Reason
				if i.logBody {
					exchange.RequestBody = truncateBody(bodyStr, i.maxBody)
				}
				// Send 403 back to client over TLS
				resp403 := &http.Response{
					StatusCode: 403,
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header:     http.Header{"Content-Type": {"text/plain"}},
					Body:       io.NopCloser(strings.NewReader("blocked by egressor: " + decision.Reason)),
				}
				resp403.Write(clientTLS)
				sess.Exchanges = append(sess.Exchanges, exchange)
				return nil
			}

			// Check content tags — hard block (e.g. NO_LLM)
			decision = i.policy.EvaluateContentTags(bodyStr)
			if !decision.Allowed {
				slog.Warn("request blocked by content tag",
					"session", sess.ID,
					"url", exchange.URL,
					"reason", decision.Reason,
				)
				exchange.StatusCode = 403
				exchange.Blocked = true
				exchange.BlockReason = decision.Reason
				if i.logBody {
					exchange.RequestBody = truncateBody(bodyStr, i.maxBody)
				}
				resp403 := &http.Response{
					StatusCode: 403,
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header:     http.Header{"Content-Type": {"text/plain"}},
					Body:       io.NopCloser(strings.NewReader("blocked by egressor: " + decision.Reason)),
				}
				resp403.Write(clientTLS)
				sess.Exchanges = append(sess.Exchanges, exchange)
				return nil
			}

			// Check content keywords — interactive approval
			kwResult := i.policy.EvaluateContentKeywords(bodyStr, paths)
			if kwResult.HasMatch {
				// Auto-blocked files from blacklist
				if len(kwResult.AutoBlocked) > 0 {
					reason := fmt.Sprintf("file %q is blacklisted (keyword %q)", kwResult.AutoBlocked[0], kwResult.MatchedKeyword)
					slog.Warn("request blocked by content keyword blacklist",
						"session", sess.ID,
						"url", exchange.URL,
						"reason", reason,
					)
					exchange.StatusCode = 403
					exchange.Blocked = true
					exchange.BlockReason = reason
					if i.logBody {
						exchange.RequestBody = truncateBody(bodyStr, i.maxBody)
					}
					resp403 := &http.Response{
						StatusCode: 403,
						ProtoMajor: 1,
						ProtoMinor: 1,
						Header:     http.Header{"Content-Type": {"text/plain"}},
						Body:       io.NopCloser(strings.NewReader("blocked by egressor: " + reason)),
					}
					resp403.Write(clientTLS)
					sess.Exchanges = append(sess.Exchanges, exchange)
					return nil
				}

				// Files needing user prompt
				if len(kwResult.NeedPrompt) > 0 && i.resolver != nil {
					promptID := fmt.Sprintf("%s-%d", sess.ID, time.Now().UnixNano())
					prompt := policy.ContentPrompt{
						ID:             promptID,
						SessionID:      sess.ID,
						URL:            exchange.URL,
						MatchedKeyword: kwResult.MatchedKeyword,
						FilePaths:      kwResult.NeedPrompt,
					}
					slog.Info("content keyword match, prompting user",
						"session", sess.ID,
						"keyword", kwResult.MatchedKeyword,
						"files", kwResult.NeedPrompt,
					)
					resp := i.resolver.PromptUser(prompt)
					switch resp.Action {
					case policy.PromptBlockOnce, policy.PromptBlockAlways:
						reason := fmt.Sprintf("user blocked: keyword %q in files %v", kwResult.MatchedKeyword, kwResult.NeedPrompt)
						exchange.StatusCode = 403
						exchange.Blocked = true
						exchange.BlockReason = reason
						if i.logBody {
							exchange.RequestBody = truncateBody(bodyStr, i.maxBody)
						}
						resp403 := &http.Response{
							StatusCode: 403,
							ProtoMajor: 1,
							ProtoMinor: 1,
							Header:     http.Header{"Content-Type": {"text/plain"}},
							Body:       io.NopCloser(strings.NewReader("blocked by egressor: " + reason)),
						}
						resp403.Write(clientTLS)
						sess.Exchanges = append(sess.Exchanges, exchange)
						return nil
					}
					// allow_once or allow_always — continue forwarding
				}
			}
		}

		if i.logBody {
			exchange.RequestBody = truncateBody(bodyStr, i.maxBody)
		}

		// Forward request to upstream
		req.URL.Scheme = "https"
		req.URL.Host = host
		req.RequestURI = "" // must be empty for http.Client
		req.Body = io.NopCloser(bytes.NewReader(reqBodyBuf.Bytes()))

		// Strip hop-by-hop headers
		stripHopByHop(req.Header)

		if err := req.Write(upstreamTLS); err != nil {
			return fmt.Errorf("writing request upstream: %w", err)
		}

		// Read response from upstream
		resp, err := http.ReadResponse(bufio.NewReader(upstreamTLS), req)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		exchange.StatusCode = resp.StatusCode
		exchange.ResponseHeaders = flattenHeaders(resp.Header)

		// Capture response body
		var respBodyBuf bytes.Buffer
		if resp.Body != nil && i.logBody {
			resp.Body = io.NopCloser(io.TeeReader(resp.Body, &limitWriter{w: &respBodyBuf, max: i.maxBody}))
		}

		// Strip hop-by-hop headers from response
		stripHopByHop(resp.Header)

		// Forward response to client
		if err := resp.Write(clientTLS); err != nil {
			resp.Body.Close()
			return fmt.Errorf("writing response to client: %w", err)
		}
		resp.Body.Close()

		if i.logBody {
			exchange.ResponseBody = truncateBody(respBodyBuf.String(), i.maxBody)
		}

		sess.Exchanges = append(sess.Exchanges, exchange)

		logAttrs := []any{
			"session", sess.ID,
			"method", exchange.Method,
			"url", exchange.URL,
			"status", exchange.StatusCode,
		}
		if len(exchange.DetectedFiles) > 0 {
			paths := make([]string, len(exchange.DetectedFiles))
			for i, f := range exchange.DetectedFiles {
				paths[i] = f.Path
			}
			logAttrs = append(logAttrs, "detected_files", paths)
		}
		slog.Info("intercepted", logAttrs...)

		// Check if connection should close
		if req.Close || resp.Close {
			return nil
		}
	}
}

// limitWriter writes up to max bytes and silently discards the rest.
type limitWriter struct {
	w       io.Writer
	max     int
	written int
}

func (lw *limitWriter) Write(p []byte) (int, error) {
	if lw.written >= lw.max {
		return len(p), nil
	}
	remaining := lw.max - lw.written
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err := lw.w.Write(p)
	lw.written += n
	return n, err
}

func flattenHeaders(h http.Header) map[string]string {
	flat := make(map[string]string, len(h))
	for k, v := range h {
		flat[k] = strings.Join(v, ", ")
	}
	return flat
}

var hopByHopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func stripHopByHop(h http.Header) {
	for _, header := range hopByHopHeaders {
		h.Del(header)
	}
}

// shouldSkipContentScan checks if the request content type matches any skip pattern.
// Supports wildcards like "image/*".
func (i *Interceptor) shouldSkipContentScan(contentType string) bool {
	if contentType == "" {
		return false
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	// Strip parameters (e.g. "text/plain; charset=utf-8" → "text/plain")
	if idx := strings.IndexByte(ct, ';'); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	for _, skip := range i.skipContentTypes {
		skip = strings.ToLower(skip)
		if strings.HasSuffix(skip, "/*") {
			// Wildcard match: "image/*" matches "image/png"
			prefix := skip[:len(skip)-1] // "image/"
			if strings.HasPrefix(ct, prefix) {
				return true
			}
		} else if ct == skip {
			return true
		}
	}
	return false
}

func truncateBody(body string, max int) string {
	if len(body) > max {
		return body[:max] + "[truncated]"
	}
	return body
}
