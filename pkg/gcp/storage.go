package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	humanize "github.com/dustin/go-humanize"
	"github.com/golang/glog"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const v1 = "v1"

// Storage is a helper layer to wrap complex Storage logic.
type Storage struct {
	projectID string

	bucket string
	prefix string

	ctx    context.Context
	client *storage.Client
}

// NewStorage returns a new Google Cloud Storage client, creating bucket if not exists.
// 'key' is a Google Developers service account JSON key.
// Create/Download the key file from https://console.cloud.google.com/apis/credentials.
func NewStorage(ctx context.Context, bucket, scope string, key []byte, prefix string) (*Storage, error) {
	// key must be JSON-format as {"project_id":...}
	credMap := make(map[string]string)
	if err := json.Unmarshal(key, &credMap); err != nil {
		return nil, err
	}
	project, ok := credMap["project_id"]
	if !ok {
		return nil, fmt.Errorf("key has no project_id")
	}

	jwt, err := google.JWTConfigFromJSON(key, scope)
	if err != nil {
		return nil, err
	}
	cli, err := storage.NewClient(ctx, option.WithTokenSource(jwt.TokenSource(ctx)))
	if err != nil {
		return nil, err
	}

	glog.Infof("creating bucket %q", bucket)
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	err = cli.Bucket(bucket).Create(cctx, project, nil)
	cancel()
	if err != nil {
		// expects; "googleapi: Error 409: You already own this bucket. Please select another name., conflict"
		// https://cloud.google.com/storage/docs/xml-api/reference-status#409conflict
		gerr, ok := err.(*googleapi.Error)
		if !ok {
			// failed to create/receive duplicate bucket
			return nil, err
		}
		if gerr.Code != 409 || gerr.Message != "You already own this bucket. Please select another name." {
			return nil, err
		}
		glog.Infof("%q already exists", bucket)
	} else {
		glog.Infof("created bucket %q", bucket)
	}

	return &Storage{projectID: project, bucket: bucket, prefix: prefix, ctx: ctx, client: cli}, nil
}

// Close closes the Client.
// Close need not be called at program exit.
func (s *Storage) Close() error {
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Put writes 'data' with 'key' as a file name in the storage.
// The actual path will be namespaced with version and prefix.
func (s *Storage) Put(key string, data []byte) error {
	glog.Infof("writing key %q (value size: %s)", key, humanize.Bytes(uint64(len(data))))
	objectName := path.Join(v1, s.prefix, key)
	wr := s.client.Bucket(s.bucket).Object(objectName).NewWriter(s.ctx)
	// TODO: set wr.ContentType?
	if _, err := wr.Write(data); err != nil {
		return err
	}
	return wr.Close()
}

// Get returns data reader for the specified 'key'.
func (s *Storage) Get(key string) (io.ReadCloser, error) {
	glog.Infof("fetching key %q", key)
	objectName := path.Join(v1, s.prefix, key)
	return s.client.Bucket(s.bucket).Object(objectName).NewReader(s.ctx)
}

// Delete deletes data for the specified 'key'.
func (s *Storage) Delete(key string) error {
	glog.Infof("deleting key %q", key)
	objectName := path.Join(v1, s.prefix, key)
	return s.client.Bucket(s.bucket).Object(objectName).Delete(s.ctx)
}

func (s *Storage) deleteBucket() error {
	return s.client.Bucket(s.bucket).Delete(s.ctx)
}

func (s *Storage) list(prefix string) (int64, []string, error) {
	glog.Infof("listing by prefix %q", prefix)

	// recursively list all "files", not directory
	pfx := path.Join(v1, prefix)
	it := s.client.Bucket(s.bucket).Objects(s.ctx, &storage.Query{Prefix: pfx})

	var attrs []*storage.ObjectAttrs
	var err error
	for {
		var attr *storage.ObjectAttrs
		attr, err = it.Next()
		if err == iterator.Done {
			err = nil
			break
		}
		if err != nil {
			return 0, nil, err
		}
		attrs = append(attrs, attr)
	}

	keys := make([]string, 0, len(attrs))
	var size int64
	for _, v := range attrs {
		name := strings.Replace(v.Name, pfx+"/", "", 1)
		keys = append(keys, name)
		size += v.Size
	}
	return size, keys, nil
}

// List lists all keys.
func (s *Storage) List() ([]string, error) {
	_, keys, err := s.list(s.prefix)
	return keys, err
}

// TotalSize returns the total size of storage.
func (s *Storage) TotalSize() (int64, error) {
	size, _, err := s.list(s.prefix)
	return size, err
}

// CopyPrefix clones data from 'from' to the receiver storage.
// Objects are assumed to be copied within the same bucket.
func (s *Storage) CopyPrefix(from string) error {
	glog.Infof("copying from %q to %q", from, s.prefix)
	_, fromKeys, err := s.list(from)
	if err != nil {
		return err
	}
	for _, key := range fromKeys {
		srcObjectName := path.Join(v1, from, key)
		srcObject := s.client.Bucket(s.bucket).Object(srcObjectName)

		// copy src to dst
		dstObjectName := path.Join(v1, s.prefix, key)
		if _, err = s.client.Bucket(s.bucket).
			Object(dstObjectName).
			CopierFrom(srcObject).
			Run(s.ctx); err != nil {
			return err
		}
	}
	return nil
}
