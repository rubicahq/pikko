package commerce

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DOKUCredentialInput struct {
	CheckoutClientID string `json:"checkout_client_id"`
	CheckoutSecret   string `json:"checkout_secret"`
	MCPClientID      string `json:"mcp_client_id"`
	MCPAPIKey        string `json:"mcp_api_key"`
}

type DOKUCredentialStatus struct {
	CheckoutClientID   string    `json:"checkout_client_id"`
	MCPClientID        string    `json:"mcp_client_id"`
	CheckoutConfigured bool      `json:"checkout_configured"`
	MCPConfigured      bool      `json:"mcp_configured"`
	Environment        string    `json:"environment"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type DOKUCredentials struct {
	StoreID          uuid.UUID
	CheckoutClientID string
	CheckoutSecret   string
	MCPClientID      string
	MCPAPIKey        string
}

func (s *Service) credentialAEAD() (cipher.AEAD, error) {
	if s == nil || s.Cfg == nil || strings.TrimSpace(s.Cfg.DOKUCredentialEncryptionKey) == "" {
		return nil, ErrDOKUUnavailable
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s.Cfg.DOKUCredentialEncryptionKey))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("%w: invalid credential encryption key", ErrDOKUUnavailable)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s *Service) encryptCredential(plain, aad string) ([]byte, error) {
	aead, err := s.credentialAEAD()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, []byte(plain), []byte(aad)), nil
}

func (s *Service) decryptCredential(ciphertext []byte, aad string) (string, error) {
	aead, err := s.credentialAEAD()
	if err != nil {
		return "", err
	}
	if len(ciphertext) < aead.NonceSize() {
		return "", ErrDOKUUnavailable
	}
	plain, err := aead.Open(nil, ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():], []byte(aad))
	if err != nil {
		return "", ErrDOKUUnavailable
	}
	return string(plain), nil
}

func (s *Service) UpsertDOKUCredentials(ctx context.Context, storeID uuid.UUID, in DOKUCredentialInput) (*DOKUCredentialStatus, error) {
	in.CheckoutClientID = strings.TrimSpace(in.CheckoutClientID)
	in.CheckoutSecret = strings.TrimSpace(in.CheckoutSecret)
	in.MCPClientID = strings.TrimSpace(in.MCPClientID)
	in.MCPAPIKey = strings.TrimSpace(in.MCPAPIKey)
	if in.CheckoutClientID == "" || in.CheckoutSecret == "" || in.MCPClientID == "" || in.MCPAPIKey == "" {
		return nil, ErrInvalidInput
	}
	checkoutCipher, err := s.encryptCredential(in.CheckoutSecret, storeID.String()+":checkout")
	if err != nil {
		return nil, err
	}
	mcpCipher, err := s.encryptCredential(in.MCPAPIKey, storeID.String()+":mcp")
	if err != nil {
		return nil, err
	}
	_, err = s.DB.Exec(ctx, `INSERT INTO store_doku_credentials (store_id,checkout_client_id,checkout_secret_ciphertext,mcp_client_id,mcp_api_key_ciphertext,environment)
		VALUES ($1,$2,$3,$4,$5,'sandbox') ON CONFLICT (store_id) DO UPDATE SET checkout_client_id=EXCLUDED.checkout_client_id,
		checkout_secret_ciphertext=EXCLUDED.checkout_secret_ciphertext,mcp_client_id=EXCLUDED.mcp_client_id,mcp_api_key_ciphertext=EXCLUDED.mcp_api_key_ciphertext,updated_at=now()`, storeID, in.CheckoutClientID, checkoutCipher, in.MCPClientID, mcpCipher)
	if err != nil {
		return nil, err
	}
	return s.GetDOKUCredentialStatus(ctx, storeID)
}

func (s *Service) GetDOKUCredentialStatus(ctx context.Context, storeID uuid.UUID) (*DOKUCredentialStatus, error) {
	var out DOKUCredentialStatus
	var checkoutCipher, mcpCipher []byte
	err := s.DB.QueryRow(ctx, `SELECT checkout_client_id,checkout_secret_ciphertext,mcp_client_id,mcp_api_key_ciphertext,environment,updated_at FROM store_doku_credentials WHERE store_id=$1`, storeID).Scan(&out.CheckoutClientID, &checkoutCipher, &out.MCPClientID, &mcpCipher, &out.Environment, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &DOKUCredentialStatus{Environment: "sandbox"}, nil
	}
	if err != nil {
		return nil, err
	}
	out.CheckoutConfigured = len(checkoutCipher) > 0
	out.MCPConfigured = len(mcpCipher) > 0
	return &out, nil
}

func (s *Service) LoadDOKUCredentials(ctx context.Context, storeID uuid.UUID) (*DOKUCredentials, error) {
	var out DOKUCredentials
	var checkoutCipher, mcpCipher []byte
	out.StoreID = storeID
	err := s.DB.QueryRow(ctx, `SELECT checkout_client_id,checkout_secret_ciphertext,mcp_client_id,mcp_api_key_ciphertext FROM store_doku_credentials WHERE store_id=$1 AND environment='sandbox'`, storeID).Scan(&out.CheckoutClientID, &checkoutCipher, &out.MCPClientID, &mcpCipher)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDOKUUnavailable
	}
	if err != nil {
		return nil, err
	}
	out.CheckoutSecret, err = s.decryptCredential(checkoutCipher, storeID.String()+":checkout")
	if err != nil {
		return nil, err
	}
	out.MCPAPIKey, err = s.decryptCredential(mcpCipher, storeID.String()+":mcp")
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) LoadDOKUCredentialsByCheckoutClient(ctx context.Context, clientID string) (*DOKUCredentials, error) {
	var storeID uuid.UUID
	err := s.DB.QueryRow(ctx, `SELECT store_id FROM store_doku_credentials WHERE checkout_client_id=$1 AND environment='sandbox'`, strings.TrimSpace(clientID)).Scan(&storeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.LoadDOKUCredentials(ctx, storeID)
}
