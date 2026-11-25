package files

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/favicon"
	"github.com/rubicahq/pikko/services/api/internal/storage"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("file not found")
	ErrForbidden    = errors.New("forbidden")
)

type File struct {
	ID               uuid.UUID `json:"id"`
	StoreID          uuid.UUID `json:"store_id"`
	Purpose          string    `json:"purpose"`
	StorageKey       string    `json:"storage_key"`
	ContentType      string    `json:"content_type"`
	SizeBytes        int64     `json:"size_bytes"`
	OriginalFilename *string   `json:"original_filename,omitempty"`
	URL              string    `json:"url,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type Service struct {
	DB      *pgxpool.Pool
	Storage storage.ObjectStore
}

func (s *Service) Upload(ctx context.Context, storeID, userID uuid.UUID, purpose, filename, declaredType string, r io.Reader, size int64) (*File, error) {
	limits, ok := storage.PurposeLimits[purpose]
	if !ok {
		return nil, ErrInvalidInput
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(r, head)
	head = head[:n]
	ct, err := storage.ValidateUpload(purpose, size, declaredType, head)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	body := io.Reader(io.MultiReader(bytes.NewReader(head), r))
	storeSize := size
	storeCT := ct
	storeName := filename

	if purpose == "favicon" {
		rest, err := io.ReadAll(io.LimitReader(r, limits.MaxBytes+1))
		if err != nil {
			return nil, fmt.Errorf("%w: unable to read upload", ErrInvalidInput)
		}
		full := append(append([]byte{}, head...), rest...)
		if int64(len(full)) > limits.MaxBytes {
			return nil, fmt.Errorf("%w: file size exceeds limit", ErrInvalidInput)
		}
		converted, outCT, err := favicon.Convert(full, ct)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		body = bytes.NewReader(converted)
		storeSize = int64(len(converted))
		storeCT = outCT
		if outCT == "image/x-icon" {
			storeName = "favicon.ico"
		} else if outCT == "image/svg+xml" && storeName == "" {
			storeName = "favicon.svg"
		}
	}

	key := storage.StoreScopedKey(storeID, purpose, storeName)
	if err := s.Storage.Put(ctx, key, body, storeCT); err != nil {
		return nil, err
	}
	var f File
	var orig *string
	if storeName != "" {
		orig = &storeName
	}
	var createdBy any
	if userID != uuid.Nil {
		createdBy = userID
	}
	err = s.DB.QueryRow(ctx, `
		INSERT INTO stored_files (store_id, purpose, storage_key, content_type, size_bytes, original_filename, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, store_id, purpose, storage_key, content_type, size_bytes, original_filename, created_at
	`, storeID, purpose, key, storeCT, storeSize, orig, createdBy).Scan(
		&f.ID, &f.StoreID, &f.Purpose, &f.StorageKey, &f.ContentType, &f.SizeBytes, &f.OriginalFilename, &f.CreatedAt,
	)
	if err != nil {
		_ = s.Storage.Delete(ctx, key)
		return nil, err
	}
	url, _ := s.Storage.SignedURL(ctx, key, time.Hour)
	f.URL = url
	return &f, nil
}

func (s *Service) GetInStore(ctx context.Context, storeID, fileID uuid.UUID) (*File, error) {
	var f File
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, purpose, storage_key, content_type, size_bytes, original_filename, created_at
		FROM stored_files WHERE id=$1 AND store_id=$2
	`, fileID, storeID).Scan(
		&f.ID, &f.StoreID, &f.Purpose, &f.StorageKey, &f.ContentType, &f.SizeBytes, &f.OriginalFilename, &f.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	url, _ := s.Storage.SignedURL(ctx, f.StorageKey, time.Hour)
	f.URL = url
	return &f, nil
}

func (s *Service) Open(ctx context.Context, storeID, fileID uuid.UUID) (*File, io.ReadCloser, error) {
	f, err := s.GetInStore(ctx, storeID, fileID)
	if err != nil {
		return nil, nil, err
	}
	if err := storage.AssertStoreScoped(storeID, f.StorageKey); err != nil {
		return nil, nil, ErrForbidden
	}
	rc, err := s.Storage.Get(ctx, f.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return f, rc, nil
}

func (s *Service) PublicURL(ctx context.Context, storeID, fileID uuid.UUID) (string, error) {
	f, err := s.GetInStore(ctx, storeID, fileID)
	if err != nil {
		return "", err
	}
	return s.Storage.SignedURL(ctx, f.StorageKey, 24*time.Hour)
}
