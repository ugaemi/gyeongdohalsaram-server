package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	allowedKeyURLPrefix = "https://static.gc.apple.com/"
	maxCertCacheSize    = 100
	certCacheTTL        = 24 * time.Hour
	certFetchTimeout    = 10 * time.Second
	maxCertSize         = 64 * 1024 // 64KB
)

// GameCenterCredential holds the data sent by the client for verification.
type GameCenterCredential struct {
	PlayerID     string `json:"player_id"`
	BundleID     string `json:"bundle_id"`
	PublicKeyURL string `json:"public_key_url"`
	Signature    string `json:"signature"`
	Salt         string `json:"salt"`
	Timestamp    uint64 `json:"timestamp"`
}

type cachedCert struct {
	cert      *x509.Certificate
	expiresAt time.Time
}

// GameCenterVerifier verifies Game Center identity signatures.
type GameCenterVerifier struct {
	httpClient         *http.Client
	certCache          map[string]*cachedCert
	mu                 sync.RWMutex
	allowedBundleIDs   map[string]bool
	timestampTolerance time.Duration
}

// NewGameCenterVerifier creates a new verifier.
func NewGameCenterVerifier(bundleIDs []string, timestampTolerance time.Duration) *GameCenterVerifier {
	allowed := make(map[string]bool, len(bundleIDs))
	for _, id := range bundleIDs {
		allowed[id] = true
	}

	return &GameCenterVerifier{
		httpClient: &http.Client{
			Timeout: certFetchTimeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return errors.New("redirects not allowed")
			},
		},
		certCache:          make(map[string]*cachedCert),
		allowedBundleIDs:   allowed,
		timestampTolerance: timestampTolerance,
	}
}

// Verify validates a Game Center identity credential.
func (v *GameCenterVerifier) Verify(ctx context.Context, cred *GameCenterCredential) error {
	if err := v.validatePublicKeyURL(cred.PublicKeyURL); err != nil {
		return fmt.Errorf("invalid public key URL: %w", err)
	}

	if err := v.validateBundleID(cred.BundleID); err != nil {
		return err
	}

	if err := v.validateTimestamp(cred.Timestamp); err != nil {
		return err
	}

	cert, err := v.fetchCertificate(ctx, cred.PublicKeyURL)
	if err != nil {
		return fmt.Errorf("failed to fetch certificate: %w", err)
	}

	signature, err := base64.StdEncoding.DecodeString(cred.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(cred.Salt)
	if err != nil {
		return fmt.Errorf("invalid salt encoding: %w", err)
	}

	payload := buildPayload(cred.PlayerID, cred.BundleID, cred.Timestamp, salt)

	hash := sha256.Sum256(payload)

	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("certificate does not contain an RSA public key")
	}

	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

func (v *GameCenterVerifier) validatePublicKeyURL(url string) error {
	if !strings.HasPrefix(url, allowedKeyURLPrefix) {
		return fmt.Errorf("URL must start with %s", allowedKeyURLPrefix)
	}
	return nil
}

func (v *GameCenterVerifier) validateBundleID(bundleID string) error {
	if len(v.allowedBundleIDs) == 0 {
		return nil
	}
	if !v.allowedBundleIDs[bundleID] {
		return fmt.Errorf("bundle ID %q is not allowed", bundleID)
	}
	return nil
}

func (v *GameCenterVerifier) validateTimestamp(timestamp uint64) error {
	if v.timestampTolerance == 0 {
		return nil
	}

	now := time.Now()
	ts := time.UnixMilli(int64(timestamp))
	diff := now.Sub(ts)
	if diff < 0 {
		diff = -diff
	}

	if diff > v.timestampTolerance {
		return fmt.Errorf("timestamp expired: difference %v exceeds tolerance %v", diff, v.timestampTolerance)
	}
	return nil
}

func (v *GameCenterVerifier) fetchCertificate(ctx context.Context, url string) (*x509.Certificate, error) {
	// Check cache
	v.mu.RLock()
	if cached, ok := v.certCache[url]; ok && time.Now().Before(cached.expiresAt) {
		v.mu.RUnlock()
		return cached.cert, nil
	}
	v.mu.RUnlock()

	// Fetch from Apple
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	certData, err := io.ReadAll(io.LimitReader(resp.Body, maxCertSize))
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Store in cache
	v.mu.Lock()
	if len(v.certCache) >= maxCertCacheSize {
		v.evictExpiredLocked()
	}
	v.certCache[url] = &cachedCert{
		cert:      cert,
		expiresAt: time.Now().Add(certCacheTTL),
	}
	v.mu.Unlock()

	return cert, nil
}

func (v *GameCenterVerifier) evictExpiredLocked() {
	now := time.Now()
	for url, cached := range v.certCache {
		if now.After(cached.expiresAt) {
			delete(v.certCache, url)
		}
	}
	// If still full after evicting expired, remove oldest
	if len(v.certCache) >= maxCertCacheSize {
		for url := range v.certCache {
			delete(v.certCache, url)
			break
		}
	}
}

func buildPayload(playerID, bundleID string, timestamp uint64, salt []byte) []byte {
	playerBytes := []byte(playerID)
	bundleBytes := []byte(bundleID)
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, timestamp)

	payload := make([]byte, 0, len(playerBytes)+len(bundleBytes)+8+len(salt))
	payload = append(payload, playerBytes...)
	payload = append(payload, bundleBytes...)
	payload = append(payload, tsBytes...)
	payload = append(payload, salt...)
	return payload
}
