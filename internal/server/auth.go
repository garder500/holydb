package server

// AWS Signature Version 4 style authentication middleware.
// This is a focused implementation sufficient for HolyDB's HTTP storage API.
// It validates the Authorization header:
//   Authorization: AWS4-HMAC-SHA256 Credential=<AccessKey>/<Date>/<Region>/<Service>/aws4_request, SignedHeaders=host;..., Signature=<hex>
// Required companion header: X-Amz-Date (format YYYYMMDD'T'HHMMSS'Z')
// Optional header: X-Amz-Content-Sha256 (payload hash). If absent, the body is read & hashed (buffered in-memory) to compute the canonical request hash.

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// context key for downstream usage (e.g., logging) of resolved access key
type ctxKey string

const (
	ctxAccessKey ctxKey = "authAccessKey"
)

// CredentialProvider resolves a secret key given an access key.
type CredentialProvider interface {
	SecretFor(accessKey string) (secret string, ok bool)
}

// RootCredentialProvider exposes a single root credential pair from env.
// HOLYDB_ROOT_USER / HOLYDB_ROOT_PASSWORD
type RootCredentialProvider struct{ user, pass string }

func NewRootCredentialProvider() *RootCredentialProvider {
	return &RootCredentialProvider{
		user: os.Getenv("HOLYDB_ROOT_USER"),
		pass: os.Getenv("HOLYDB_ROOT_PASSWORD"),
	}
}

func (p *RootCredentialProvider) SecretFor(accessKey string) (string, bool) {
	if p.user == "" || p.pass == "" {
		return "", false
	}
	if accessKey == p.user {
		return p.pass, true
	}
	return "", false
}

// AuthConfig controls middleware behavior.
type AuthConfig struct {
	Region      string        // default from HOLYDB_REGION (us-east-1)
	Service     string        // fixed: holydb unless overridden
	ClockSkew   time.Duration // acceptable skew (default 5m)
	Credentials CredentialProvider
	RequireAuth bool // if false, bypass when no Authorization header
}

func defaultAuthConfig() AuthConfig {
	region := os.Getenv("HOLYDB_REGION")
	if region == "" {
		region = "us-east-1"
	}
	return AuthConfig{
		Region:      region,
		Service:     "holydb",
		ClockSkew:   5 * time.Minute,
		Credentials: NewRootCredentialProvider(),
		RequireAuth: true,
	}
}

// AuthMiddleware returns an HTTP middleware that validates AWS v4 signatures.
func AuthMiddleware(cfg *AuthConfig) func(http.Handler) http.Handler {
	c := defaultAuthConfig()
	if cfg != nil {
		if cfg.Region != "" {
			c.Region = cfg.Region
		}
		if cfg.Service != "" {
			c.Service = cfg.Service
		}
		if cfg.ClockSkew != 0 {
			c.ClockSkew = cfg.ClockSkew
		}
		if cfg.Credentials != nil {
			c.Credentials = cfg.Credentials
		}
		c.RequireAuth = cfg.RequireAuth
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if authz == "" {
				if c.RequireAuth {
					http.Error(w, "missing Authorization header", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			if !strings.HasPrefix(authz, "AWS4-HMAC-SHA256 ") {
				http.Error(w, "invalid auth scheme", http.StatusUnauthorized)
				return
			}
			parsed, err := parseAuthHeader(authz[len("AWS4-HMAC-SHA256 "):])
			if err != nil {
				http.Error(w, "malformed Authorization", http.StatusUnauthorized)
				return
			}
			accessKey := parsed.CredentialParts[0]
			if len(parsed.CredentialParts) != 5 || parsed.CredentialParts[4] != "aws4_request" {
				http.Error(w, "invalid credential scope", http.StatusUnauthorized)
				return
			}
			datePart, region, service := parsed.CredentialParts[1], parsed.CredentialParts[2], parsed.CredentialParts[3]
			if region != c.Region || service != c.Service {
				http.Error(w, "invalid region/service", http.StatusUnauthorized)
				return
			}
			secret, ok := c.Credentials.SecretFor(accessKey)
			if !ok {
				http.Error(w, "unknown access key", http.StatusUnauthorized)
				return
			}
			amzDate := r.Header.Get("X-Amz-Date")
			if amzDate == "" {
				http.Error(w, "missing X-Amz-Date", http.StatusUnauthorized)
				return
			}
			t, err := time.Parse("20060102T150405Z", amzDate)
			if err != nil {
				http.Error(w, "bad X-Amz-Date", http.StatusUnauthorized)
				return
			}
			if absDuration(time.Since(t)) > c.ClockSkew {
				http.Error(w, "request expired", http.StatusUnauthorized)
				return
			}
			if !strings.HasPrefix(amzDate, datePart) {
				http.Error(w, "date scope mismatch", http.StatusUnauthorized)
				return
			}
			// Build canonical request
			payloadHash, restoredBody, err := ensurePayloadHash(r)
			if err != nil {
				http.Error(w, "payload hash error", http.StatusUnauthorized)
				return
			}
			if restoredBody != nil {
				r.Body = restoredBody
			}
			canonicalReq, err := buildCanonicalRequest(r, parsed.SignedHeaders, payloadHash)
			if err != nil {
				http.Error(w, "canonical request error", http.StatusUnauthorized)
				return
			}
			hashCR := sha256Hex([]byte(canonicalReq))
			stringToSign := strings.Join([]string{
				"AWS4-HMAC-SHA256",
				amzDate,
				strings.Join(parsed.CredentialParts[1:], "/"),
				hashCR,
			}, "\n")
			// Derive signing key
			kDate := hmacSHA256([]byte("AWS4"+secret), []byte(datePart))
			kRegion := hmacSHA256(kDate, []byte(region))
			kService := hmacSHA256(kRegion, []byte(service))
			kSigning := hmacSHA256(kService, []byte("aws4_request"))
			expectedSig := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))
			if !hmac.Equal([]byte(expectedSig), []byte(parsed.Signature)) {
				http.Error(w, "signature mismatch", http.StatusUnauthorized)
				return
			}
			// success
			ctx := context.WithValue(r.Context(), ctxAccessKey, accessKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// authHeader holds parsed parts.
type authHeader struct {
	CredentialParts []string
	SignedHeaders   []string
	Signature       string
}

func parseAuthHeader(rest string) (*authHeader, error) {
	// rest looks like: Credential=AKIA.../date/region/service/aws4_request, SignedHeaders=host;x-amz-date;..., Signature=abcdef
	parts := splitCommaAware(rest)
	ah := &authHeader{}
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := kv[0]
		v := kv[1]
		switch k {
		case "Credential":
			ah.CredentialParts = strings.Split(v, "/")
		case "SignedHeaders":
			ah.SignedHeaders = strings.Split(v, ";")
		case "Signature":
			ah.Signature = v
		}
	}
	if len(ah.CredentialParts) == 0 || len(ah.SignedHeaders) == 0 || ah.Signature == "" {
		return nil, errors.New("missing fields")
	}
	return ah, nil
}

func splitCommaAware(s string) []string {
	// Values themselves don't contain commas in SigV4 header, so simple split.
	return strings.Split(s, ",")
}

func ensurePayloadHash(r *http.Request) (string, io.ReadCloser, error) {
	if h := r.Header.Get("X-Amz-Content-Sha256"); h != "" {
		return h, nil, nil
	}
	if r.Body == nil {
		return sha256Hex([]byte{}), nil, nil
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return "", nil, err
	}
	_ = r.Body.Close()
	hash := sha256Hex(data)
	return hash, io.NopCloser(bytes.NewReader(data)), nil
}

func buildCanonicalRequest(r *http.Request, signedHeaders []string, payloadHash string) (string, error) {
	// Method
	method := r.Method
	// Canonical URI
	uri := canonicalURI(r.URL.EscapedPath())
	// Canonical Query String
	qs := canonicalQueryString(r.URL.RawQuery)
	// Canonical Headers
	headers, err := canonicalHeaders(r, signedHeaders)
	if err != nil {
		return "", err
	}
	// Signed headers (lowercase already ensured in canonicalHeaders)
	sh := strings.Join(signedHeaders, ";")
	return strings.Join([]string{method, uri, qs, headers, sh, payloadHash}, "\n"), nil
}

func canonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	// Ensure no double encoding; split on '/'
	segs := strings.Split(path, "/")
	for i, s := range segs {
		segs[i] = awsURLEncode(decodePercents(s), false)
	}
	return strings.Join(segs, "/")
}

func decodePercents(s string) string {
	u, err := url.PathUnescape(s)
	if err != nil {
		return s
	}
	return u
}

func canonicalQueryString(raw string) string {
	if raw == "" {
		return ""
	}
	v, _ := url.ParseQuery(raw)
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	first := true
	for _, k := range keys {
		vals := v[k]
		sort.Strings(vals)
		for _, val := range vals {
			if !first {
				b.WriteByte('&')
			} else {
				first = false
			}
			b.WriteString(awsURLEncode(k, true))
			b.WriteByte('=')
			b.WriteString(awsURLEncode(val, true))
		}
	}
	return b.String()
}

func canonicalHeaders(r *http.Request, signed []string) (string, error) {
	// Ensure headers list is lowercase and sorted as provided order matters only for matching; we'll trust header list order but AWS expects alphabetical – keep consistent by sorting.
	for i, h := range signed {
		signed[i] = strings.ToLower(h)
	}
	sort.Strings(signed)
	var b strings.Builder
	for _, name := range signed {
		values := r.Header[http.CanonicalHeaderKey(name)]
		if len(values) == 0 {
			// some headers like host might not be in Header map; handle host separately
			if name == "host" {
				v := r.Host
				if v == "" {
					return "", errors.New("missing host header")
				}
				b.WriteString(name)
				b.WriteByte(':')
				b.WriteString(strings.TrimSpace(v))
				b.WriteByte('\n')
				continue
			}
			return "", errors.New("missing signed header: " + name)
		}
		// Collapse values with comma+space per AWS spec (trim internal spaces)
		collapsed := make([]string, 0, len(values))
		for _, v := range values {
			collapsed = append(collapsed, strings.Join(strings.Fields(v), " "))
		}
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(strings.Join(collapsed, ","))
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func awsURLEncode(s string, encodeSlash bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		if c == '/' && !encodeSlash {
			b.WriteByte('/')
			continue
		}
		b.WriteString("%")
		b.WriteString(strings.ToUpper(hex.EncodeToString([]byte{c})))
	}
	return b.String()
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
