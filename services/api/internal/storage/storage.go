package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ObjectStore abstracts file storage (PRD §31). Local for dev; S3-compatible later.
type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type LocalStore struct {
	Root string
}

func NewLocal(root string) (*LocalStore, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalStore{Root: root}, nil
}

func (s *LocalStore) path(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid storage key")
	}
	return filepath.Join(s.Root, clean), nil
}

func (s *LocalStore) Put(_ context.Context, key string, r io.Reader, _ string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s *LocalStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	return os.Remove(p)
}

func (s *LocalStore) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	// Local adapter returns a relative path marker; reverse proxy serves private files via API.
	return "/api/v1/files/" + strings.TrimPrefix(key, "/"), nil
}

// S3Store is a stub for future S3-compatible backends.
type S3Store struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}

func NewS3(endpoint, bucket, access, secret string) *S3Store {
	return &S3Store{Endpoint: endpoint, Bucket: bucket, AccessKey: access, SecretKey: secret}
}

func (s *S3Store) Put(context.Context, string, io.Reader, string) error {
	return fmt.Errorf("S3 storage not configured in this environment")
}
func (s *S3Store) Get(context.Context, string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("S3 storage not configured in this environment")
}
func (s *S3Store) Delete(context.Context, string) error {
	return fmt.Errorf("S3 storage not configured in this environment")
}
func (s *S3Store) SignedURL(context.Context, string, time.Duration) (string, error) {
	return "", fmt.Errorf("S3 storage not configured in this environment")
}

func NewFromDriver(driver, localPath, endpoint, bucket, access, secret string) (ObjectStore, error) {
	switch strings.ToLower(driver) {
	case "local", "":
		return NewLocal(localPath)
	case "s3":
		return NewS3(endpoint, bucket, access, secret), nil
	default:
		return nil, fmt.Errorf("unknown STORAGE_DRIVER %q", driver)
	}
}
