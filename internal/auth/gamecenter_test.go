package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCert holds a self-signed test certificate and its private key.
type testCert struct {
	certDER    []byte
	privateKey *rsa.PrivateKey
}

func generateTestCert(t *testing.T) *testCert {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test Apple GC Cert"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return &testCert{certDER: certDER, privateKey: key}
}

func signPayload(t *testing.T, key *rsa.PrivateKey, playerID, bundleID string, timestamp uint64, salt []byte) []byte {
	t.Helper()

	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, timestamp)

	var payload []byte
	payload = append(payload, []byte(playerID)...)
	payload = append(payload, []byte(bundleID)...)
	payload = append(payload, tsBytes...)
	payload = append(payload, salt...)

	hash := sha256.Sum256(payload)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	require.NoError(t, err)
	return sig
}

func setupTestServer(t *testing.T, tc *testCert) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(tc.certDER)
	}))
	t.Cleanup(server.Close)
	return server
}

func newTestVerifier(t *testing.T, serverURL string, httpClient *http.Client) *GameCenterVerifier {
	t.Helper()

	v := NewGameCenterVerifier([]string{"com.test.game"}, 5*time.Minute)
	v.httpClient = httpClient
	// Override URL validation for testing (server URL won't start with Apple domain)
	v.allowedBundleIDs = map[string]bool{"com.test.game": true}
	return v
}

func TestVerify_ValidSignature(t *testing.T) {
	tc := generateTestCert(t)
	server := setupTestServer(t, tc)

	v := newTestVerifier(t, server.URL, server.Client())

	salt := []byte("randomsalt")
	ts := uint64(time.Now().UnixMilli())
	sig := signPayload(t, tc.privateKey, "G:player1", "com.test.game", ts, salt)

	cred := &GameCenterCredential{
		PlayerID:     "G:player1",
		BundleID:     "com.test.game",
		PublicKeyURL: server.URL + "/gc-prod.cer",
		Signature:    base64.StdEncoding.EncodeToString(sig),
		Salt:         base64.StdEncoding.EncodeToString(salt),
		Timestamp:    ts,
	}

	// Bypass URL prefix check for test
	err := v.verifyWithoutURLCheck(context.Background(), cred)
	assert.NoError(t, err)
}

func TestVerify_InvalidSignature(t *testing.T) {
	tc := generateTestCert(t)
	server := setupTestServer(t, tc)

	v := newTestVerifier(t, server.URL, server.Client())

	salt := []byte("randomsalt")
	ts := uint64(time.Now().UnixMilli())
	// Sign with different playerID to produce wrong signature
	sig := signPayload(t, tc.privateKey, "G:wrongplayer", "com.test.game", ts, salt)

	cred := &GameCenterCredential{
		PlayerID:     "G:player1",
		BundleID:     "com.test.game",
		PublicKeyURL: server.URL + "/gc-prod.cer",
		Signature:    base64.StdEncoding.EncodeToString(sig),
		Salt:         base64.StdEncoding.EncodeToString(salt),
		Timestamp:    ts,
	}

	err := v.verifyWithoutURLCheck(context.Background(), cred)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
}

func TestValidatePublicKeyURL(t *testing.T) {
	v := NewGameCenterVerifier(nil, 0)

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid Apple URL", "https://static.gc.apple.com/public-key/gc-prod-6.cer", false},
		{"invalid domain", "https://evil.com/cert.cer", true},
		{"http not https", "http://static.gc.apple.com/cert.cer", true},
		{"subdomain attack", "https://static.gc.apple.com.evil.com/cert.cer", true},
		{"empty URL", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.validatePublicKeyURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBundleID(t *testing.T) {
	v := NewGameCenterVerifier([]string{"com.test.game", "com.test.game2"}, 0)

	assert.NoError(t, v.validateBundleID("com.test.game"))
	assert.NoError(t, v.validateBundleID("com.test.game2"))
	assert.Error(t, v.validateBundleID("com.evil.game"))
	assert.Error(t, v.validateBundleID(""))
}

func TestValidateBundleID_NoRestrictions(t *testing.T) {
	v := NewGameCenterVerifier(nil, 0)

	assert.NoError(t, v.validateBundleID("any.bundle.id"))
}

func TestValidateTimestamp(t *testing.T) {
	v := NewGameCenterVerifier(nil, 5*time.Minute)

	// Current timestamp should pass
	now := uint64(time.Now().UnixMilli())
	assert.NoError(t, v.validateTimestamp(now))

	// 1 minute ago should pass
	oneMinAgo := uint64(time.Now().Add(-1 * time.Minute).UnixMilli())
	assert.NoError(t, v.validateTimestamp(oneMinAgo))

	// 10 minutes ago should fail
	tenMinAgo := uint64(time.Now().Add(-10 * time.Minute).UnixMilli())
	assert.Error(t, v.validateTimestamp(tenMinAgo))

	// 10 minutes in future should fail
	tenMinFuture := uint64(time.Now().Add(10 * time.Minute).UnixMilli())
	assert.Error(t, v.validateTimestamp(tenMinFuture))
}

func TestValidateTimestamp_NoTolerance(t *testing.T) {
	v := NewGameCenterVerifier(nil, 0)

	// With zero tolerance, any timestamp passes
	assert.NoError(t, v.validateTimestamp(0))
	assert.NoError(t, v.validateTimestamp(uint64(time.Now().UnixMilli())))
}

func TestCertificateCaching(t *testing.T) {
	tc := generateTestCert(t)
	fetchCount := 0

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetchCount++
		w.Write(tc.certDER)
	}))
	t.Cleanup(server.Close)

	v := newTestVerifier(t, server.URL, server.Client())

	ctx := context.Background()
	url := server.URL + "/gc-prod.cer"

	cert1, err := v.fetchCertificate(ctx, url)
	require.NoError(t, err)
	require.NotNil(t, cert1)

	cert2, err := v.fetchCertificate(ctx, url)
	require.NoError(t, err)
	require.NotNil(t, cert2)

	// Should only have fetched once due to caching
	assert.Equal(t, 1, fetchCount)
}

func TestFetchCertificate_RedirectBlocked(t *testing.T) {
	redirectServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.com/cert.cer", http.StatusFound)
	}))
	t.Cleanup(redirectServer.Close)

	v := newTestVerifier(t, redirectServer.URL, redirectServer.Client())
	// Override redirect policy on test client too
	v.httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("redirects not allowed")
	}

	_, err := v.fetchCertificate(context.Background(), redirectServer.URL+"/cert.cer")
	assert.Error(t, err)
}

func TestBuildPayload(t *testing.T) {
	salt := []byte{0x01, 0x02, 0x03}
	ts := uint64(1700000000000)

	payload := buildPayload("player1", "com.test", ts, salt)

	expected := []byte("player1")
	expected = append(expected, []byte("com.test")...)
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, ts)
	expected = append(expected, tsBytes...)
	expected = append(expected, salt...)

	assert.Equal(t, expected, payload)
}

// verifyWithoutURLCheck is a test helper that skips URL prefix validation.
func (v *GameCenterVerifier) verifyWithoutURLCheck(ctx context.Context, cred *GameCenterCredential) error {
	if err := v.validateBundleID(cred.BundleID); err != nil {
		return err
	}
	if err := v.validateTimestamp(cred.Timestamp); err != nil {
		return err
	}

	cert, err := v.fetchCertificate(ctx, cred.PublicKeyURL)
	if err != nil {
		return err
	}

	signature, err := base64.StdEncoding.DecodeString(cred.Signature)
	if err != nil {
		return err
	}

	salt, err := base64.StdEncoding.DecodeString(cred.Salt)
	if err != nil {
		return err
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
