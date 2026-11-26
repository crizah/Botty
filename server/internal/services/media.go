package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	db "server/internal/db/sqlc"
)

// imageTypes are the only image types we keep, checked against the file's real bytes.
var imageTypes = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}

// AllowedImageType reports whether a (claimed) MIME type is one we store.
func AllowedImageType(mime string) bool { _, ok := imageTypes[mime]; return ok }

// BlobStore keeps image bytes by name. DiskStore is the only implementation; an S3 one can replace it.
type BlobStore interface {
	Put(name string, data []byte) error
	Open(name string) (*os.File, error)
	Remove(name string)
}

// DiskStore saves files in one directory, private to the server process.
type DiskStore struct{ dir string }

func NewDiskStore(dir string) (*DiskStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &DiskStore{dir: dir}, nil
}

// path only accepts plain file names, so a stored name can never point outside the directory.
func (d *DiskStore) path(name string) (string, error) {
	if name == "" || name != filepath.Base(name) {
		return "", errors.New("bad file name")
	}
	return filepath.Join(d.dir, name), nil
}

func (d *DiskStore) Put(name string, data []byte) error {
	p, err := d.path(name)
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (d *DiskStore) Open(name string) (*os.File, error) {
	p, err := d.path(name)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (d *DiskStore) Remove(name string) {
	if p, err := d.path(name); err == nil {
		_ = os.Remove(p)
	}
}

// MediaService stores report images and serves them to project members.
type MediaService struct {
	db       *sql.DB
	Queries  *db.Queries
	store    BlobStore
	MaxBytes int64
}

func NewMediaService(dbConn *sql.DB, store BlobStore, maxBytes int64) *MediaService {
	return &MediaService{db: dbConn, Queries: db.New(dbConn), store: store, MaxBytes: maxBytes}
}

// checkImage returns the real MIME type of data, or an error if it is too big or not an allowed image.
func checkImage(data []byte, max int64) (string, error) {
	if int64(len(data)) > max {
		return "", fmt.Errorf("image is %d bytes, over the %d limit", len(data), max)
	}
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	mime := http.DetectContentType(head)
	if !AllowedImageType(mime) {
		return "", fmt.Errorf("not an allowed image type (%s)", mime)
	}
	return mime, nil
}

// Save checks the bytes, writes them under a random name and links them to the report.
func (s *MediaService) Save(ctx context.Context, tc TagContext, reportID, messageID string, data []byte) error {
	mime, err := checkImage(data, s.MaxBytes)
	if err != nil {
		return err
	}
	name := newID() + imageTypes[mime]
	if err := s.store.Put(name, data); err != nil {
		return err
	}
	if err := s.Queries.CreateMedia(ctx, db.CreateMediaParams{ID: newID(), ReportID: reportID, MessageID: messageID, File: name, Mime: mime, Size: int64(len(data))}); err != nil {
		s.store.Remove(name)
		return err
	}
	return nil
}

// Open returns the image file for a project member (viewer or above).
func (s *MediaService) Open(ctx context.Context, userID, mediaID string) (*os.File, string, error) {
	m, err := s.Queries.GetMediaFile(ctx, mediaID)
	if err != nil {
		return nil, "", notFoundOr(err, "image not found")
	}
	if _, err := requireProject(ctx, s.Queries, m.ProjectID, userID, levelViewer); err != nil {
		return nil, "", NotFound("image not found") // don't reveal that it exists
	}
	f, err := s.store.Open(m.File)
	if err != nil {
		return nil, "", NotFound("image not found")
	}
	return f, m.Mime, nil
}
