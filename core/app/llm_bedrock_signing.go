package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func signBedrockRequest(req *http.Request, payload []byte) error {
	cfg, err := bedrockSigningConfigForHost(req.URL.Hostname())
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(payload)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if cfg.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", cfg.SessionToken)
	}
	signedHeaders := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	canonicalHeaders := "host:" + req.URL.Host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n"
	if token := req.Header.Get("X-Amz-Security-Token"); token != "" {
		signedHeaders = []string{"host", "x-amz-content-sha256", "x-amz-date", "x-amz-security-token"}
		canonicalHeaders += "x-amz-security-token:" + strings.TrimSpace(token) + "\n"
	}
	signedHeadersValue := strings.Join(signedHeaders, ";")
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeadersValue,
		payloadHash,
	}, "\n")
	scope := strings.Join([]string{dateStamp, cfg.Region, "bedrock", "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := fmt.Sprintf("%x", hmacSHA256(awsSigningKey(cfg.SecretKey, dateStamp, cfg.Region, "bedrock"), []byte(stringToSign)))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", cfg.AccessKey, scope, signedHeadersValue, signature))
	return nil
}

func bedrockSigningConfigForHost(host string) (bedrockSigningConfig, error) {
	cfg := bedrockSigningConfig{
		AccessKey:    lookupEnv("AWS_ACCESS_KEY_ID"),
		SecretKey:    lookupEnv("AWS_SECRET_ACCESS_KEY"),
		SessionToken: lookupEnv("AWS_SESSION_TOKEN"),
		Region:       firstNonEmpty(lookupEnv("AWS_REGION"), lookupEnv("AWS_DEFAULT_REGION"), bedrockRegionFromHost(host)),
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return bedrockSigningConfig{}, errors.New("AWS credentials are not configured: set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
	}
	if cfg.Region == "" {
		return bedrockSigningConfig{}, errors.New("AWS region is not configured: set AWS_REGION or use a regional Bedrock runtime URL")
	}
	return cfg, nil
}

func bedrockRegionFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if !strings.HasPrefix(host, "bedrock-runtime.") || !strings.HasSuffix(host, ".amazonaws.com") {
		return ""
	}
	region := strings.TrimSuffix(strings.TrimPrefix(host, "bedrock-runtime."), ".amazonaws.com")
	if strings.Contains(region, ".") {
		return ""
	}
	return region
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])
}

func awsSigningKey(secret string, date string, region string, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key []byte, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}
