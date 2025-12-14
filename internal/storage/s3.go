package storage

import (
	"context"
	"errors"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type S3Storage struct {
	region       string
	bucket       string
	endpoint     string
	accessKey    string
	secretKey    string
	sessionToken string
	client       *http.Client
}

func NewS3Storage(region, bucket, endpoint, ak, sk, st string) *S3Storage {
	if endpoint == "" {
		endpoint = "https://" + bucket + ".s3." + region + ".amazonaws.com"
	} else {
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}
		endpoint = strings.TrimRight(endpoint, "/")
	}
	return &S3Storage{
		region:       region,
		bucket:       bucket,
		endpoint:     endpoint,
		accessKey:    ak,
		secretKey:    sk,
		sessionToken: st,
		client:       &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader) error {
	if s.accessKey == "" || s.secretKey == "" {
		return errors.New("s3 credentials not set")
	}
	u := s.objectURL(key)
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	payloadHash := hex.EncodeToString(sum[:])
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	s.setCommonHeaders(req, payloadHash)
	if err := s.sign(req, payloadHash); err != nil {
		return err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusNoContent {
		b2, _ := io.ReadAll(res.Body)
		return errors.New(string(b2))
	}
	return nil
}

func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if s.accessKey == "" || s.secretKey == "" {
		return nil, errors.New("s3 credentials not set")
	}
	u := s.objectURL(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	empty := sha256.Sum256(nil)
	payloadHash := hex.EncodeToString(empty[:])
	s.setCommonHeaders(req, payloadHash)
	if err := s.sign(req, payloadHash); err != nil {
		return nil, err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		b2, _ := io.ReadAll(res.Body)
		res.Body.Close()
		return nil, errors.New(string(b2))
	}
	return res.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if s.accessKey == "" || s.secretKey == "" {
		return errors.New("s3 credentials not set")
	}
	u := s.objectURL(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	empty := sha256.Sum256(nil)
	payloadHash := hex.EncodeToString(empty[:])
	s.setCommonHeaders(req, payloadHash)
	if err := s.sign(req, payloadHash); err != nil {
		return err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		b2, _ := io.ReadAll(res.Body)
		return errors.New(string(b2))
	}
	return nil
}

func (s *S3Storage) objectURL(key string) string {
	key = strings.TrimLeft(key, "/")
	if strings.Contains(s.endpoint, "amazonaws.com") && strings.Contains(s.endpoint, s.bucket) {
		return s.endpoint + "/" + url.PathEscape(key)
	}
	return s.endpoint + "/" + s.bucket + "/" + url.PathEscape(key)
}

func (s *S3Storage) setCommonHeaders(req *http.Request, payloadHash string) {
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if s.sessionToken != "" {
		req.Header.Set("x-amz-security-token", s.sessionToken)
	}
}

func (s *S3Storage) sign(req *http.Request, payloadHash string) error {
	t := req.Header.Get("x-amz-date")
	if t == "" {
		return errors.New("missing amz date")
	}
	ymd := t[:8]
	host := req.URL.Host
	req.Header.Set("host", host)
	var headers []struct{ k, v string }
	headers = append(headers, struct{ k, v string }{"host", host})
	headers = append(headers, struct{ k, v string }{"x-amz-content-sha256", payloadHash})
	headers = append(headers, struct{ k, v string }{"x-amz-date", t})
	signed := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	if s.sessionToken != "" {
		headers = append(headers, struct{ k, v string }{"x-amz-security-token", s.sessionToken})
		signed = append(signed, "x-amz-security-token")
	}
	var canon strings.Builder
	for _, h := range headers {
		canon.WriteString(h.k)
		canon.WriteString(":")
		canon.WriteString(strings.TrimSpace(h.v))
		canon.WriteString("\n")
	}
	canonReq := strings.Join([]string{
		req.Method,
		canonicalURI(req.URL.Path),
		canonicalQuery(req.URL.RawQuery),
		canon.String(),
		strings.Join(signed, ";"),
		payloadHash,
	}, "\n")
	scope := strings.Join([]string{ymd, s.region, "s3", "aws4_request"}, "/")
	sum := sha256.Sum256([]byte(canonReq))
	toSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		t,
		scope,
		hex.EncodeToString(sum[:]),
	}, "\n")
	kDate := hmacSHA256([]byte("AWS4"+s.secretKey), []byte(ymd))
	kRegion := hmacSHA256(kDate, []byte(s.region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	signature := hmacSHA256Hex(kSigning, []byte(toSign))
	auth := "AWS4-HMAC-SHA256 Credential=" + s.accessKey + "/" + scope + ", SignedHeaders=" + strings.Join(signed, ";") + ", Signature=" + signature
	req.Header.Set("Authorization", auth)
	return nil
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func hmacSHA256Hex(key, data []byte) string {
	return hex.EncodeToString(hmacSHA256(key, data))
}

func canonicalURI(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func canonicalQuery(q string) string {
	return q
}
