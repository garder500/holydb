package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper to compute sha256 hex
func sha256HexBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func hmacSHA256Bytes(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// buildSigV4 builds headers (Authorization, X-Amz-Date, X-Amz-Content-Sha256) for a request.
func buildSigV4(req *http.Request, accessKey, secret string, body []byte, amzTime time.Time) {
	region := "us-east-1"
	service := "holydb"
	amzDate := amzTime.UTC().Format("20060102T150405Z")
	datePart := amzDate[:8]
	payloadHash := sha256HexBytes(body)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("X-Amz-Date", amzDate)
	// Signed headers list
	signed := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	// Canonical headers (sorted already lexicographically)
	// host header uses req.Host
	var headersBuilder strings.Builder
	headersBuilder.WriteString("host:")
	headersBuilder.WriteString(req.Host)
	headersBuilder.WriteByte('\n')
	headersBuilder.WriteString("x-amz-content-sha256:")
	headersBuilder.WriteString(payloadHash)
	headersBuilder.WriteByte('\n')
	headersBuilder.WriteString("x-amz-date:")
	headersBuilder.WriteString(amzDate)
	headersBuilder.WriteByte('\n')
	canonicalHeaders := headersBuilder.String()
	signedHdrs := strings.Join(signed, ";")
	// Canonical request replicating server logic (note double newline before signed headers because headers string ends with \n and we join with '\n').
	canonicalReq := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		"", // query string empty
		canonicalHeaders,
		signedHdrs,
		payloadHash,
	}, "\n")
	hashCR := sha256HexBytes([]byte(canonicalReq))
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		strings.Join([]string{datePart, region, service, "aws4_request"}, "/"),
		hashCR,
	}, "\n")
	kDate := hmacSHA256Bytes([]byte("AWS4"+secret), []byte(datePart))
	kRegion := hmacSHA256Bytes(kDate, []byte(region))
	kService := hmacSHA256Bytes(kRegion, []byte(service))
	kSigning := hmacSHA256Bytes(kService, []byte("aws4_request"))
	sig := hex.EncodeToString(hmacSHA256Bytes(kSigning, []byte(stringToSign)))
	authHeader := strings.Join([]string{
		"AWS4-HMAC-SHA256 Credential=" + accessKey + "/" + strings.Join([]string{datePart, region, service, "aws4_request"}, "/"),
		"SignedHeaders=" + signedHdrs,
		"Signature=" + sig,
	}, ",")
	req.Header.Set("Authorization", authHeader)
}

func TestAuthMiddlewareSigV4(t *testing.T) {
	// Setup credentials
	oldUser := os.Getenv("HOLYDB_ROOT_USER")
	oldPass := os.Getenv("HOLYDB_ROOT_PASSWORD")
	defer os.Setenv("HOLYDB_ROOT_USER", oldUser)
	defer os.Setenv("HOLYDB_ROOT_PASSWORD", oldPass)
	if err := os.Setenv("HOLYDB_ROOT_USER", "test"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("HOLYDB_ROOT_PASSWORD", "secret123"); err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	srv := New(Config{Addr: ":0", Root: tmp}) // handler only

	// 1. Missing Authorization should 401
	req := httptest.NewRequest(http.MethodPut, "/v1/storage/bkt/obj1", strings.NewReader("hello"))
	req.Host = "localhost"
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing Authorization, got %d", rr.Code)
	}

	// 2. Valid signature -> 201 Created
	body := []byte("world")
	req2 := httptest.NewRequest(http.MethodPut, "/v1/storage/bkt/obj2", strings.NewReader(string(body)))
	req2.Host = "localhost"
	buildSigV4(req2, "test", "secret123", body, time.Now())
	rr2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid signature, got %d", rr2.Code)
	}
	// Verify file created
	if _, err := os.Stat(filepath.Join(tmp, "bkt", "obj2", "part.1")); err != nil {
		t.Fatalf("expected object written: %v", err)
	}

	// 3. Bad signature -> 401
	bad := httptest.NewRequest(http.MethodPut, "/v1/storage/bkt/obj3", strings.NewReader("bad"))
	bad.Host = "localhost"
	buildSigV4(bad, "test", "secret123", []byte("bad"), time.Now())
	// Corrupt signature
	bad.Header.Set("Authorization", bad.Header.Get("Authorization")+"00")
	rr3 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr3, bad)
	if rr3.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad signature, got %d", rr3.Code)
	}
}
