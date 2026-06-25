package s3_storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"databasus-backend/internal/util/encryption"
)

const (
	s3ConnectTimeout      = 30 * time.Second
	s3ResponseTimeout     = 30 * time.Second
	s3IdleConnTimeout     = 90 * time.Second
	s3TLSHandshakeTimeout = 30 * time.Second
	s3DeleteTimeout       = 30 * time.Second

	// defaultInnerPartSize is the multipart part we feed S3 for one object. 16 MiB balances
	// memory against request count and creates backpressure to pg_dump: we keep one part in
	// flight and wait for S3 to confirm it before reading the next, so pipeline memory does not
	// grow with the database size (ADR-0004).
	defaultInnerPartSize = 16 * 1024 * 1024

	// defaultMaxPartsPerObject caps inner parts per object; once reached we roll to the next
	// object. Effective object size = defaultMaxPartsPerObject * defaultInnerPartSize ~= 15.6 GiB.
	// S3 multipart is bounded by part_size * part_count_limit and that limit is provider-set
	// (AWS 10000, Scaleway and many S3-compatibles 1000). Because we stream dumps of unknown size,
	// we write blind to the smallest common limit so one backup can span any provider without
	// knowing its real cap; the object count, not memory, scales with the database size.
	defaultMaxPartsPerObject = 1000

	manifestSuffix  = ".parts"
	manifestVersion = 1
)

type S3Storage struct {
	StorageID   uuid.UUID `json:"storageId"   gorm:"primaryKey;type:uuid;column:storage_id"`
	S3Bucket    string    `json:"s3Bucket"    gorm:"not null;type:text;column:s3_bucket"`
	S3Region    string    `json:"s3Region"    gorm:"not null;type:text;column:s3_region"`
	S3AccessKey string    `json:"s3AccessKey" gorm:"not null;type:text;column:s3_access_key"`
	S3SecretKey string    `json:"s3SecretKey" gorm:"not null;type:text;column:s3_secret_key"`
	S3Endpoint  string    `json:"s3Endpoint"  gorm:"type:text;column:s3_endpoint"`

	S3Prefix                string         `json:"s3Prefix"                gorm:"type:text;column:s3_prefix"`
	S3UseVirtualHostedStyle bool           `json:"s3UseVirtualHostedStyle" gorm:"default:false;column:s3_use_virtual_hosted_style"`
	SkipTLSVerify           bool           `json:"skipTLSVerify"           gorm:"default:false;column:skip_tls_verify"`
	S3StorageClass          S3StorageClass `json:"s3StorageClass"          gorm:"type:text;column:s3_storage_class;default:''"`

	// Test seams: rolling at the production 15.6 GiB object boundary would need ~15 GiB per case,
	// so integration tests shrink the inner part and per-object part count to exercise object
	// rollover on a few tens of MiB. Unexported, so gorm and JSON ignore them; zero means default.
	innerPartSizeOverride int
	maxPartsOverride      int
}

func (s *S3Storage) TableName() string {
	return "s3_storages"
}

func (s *S3Storage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
	file io.Reader,
) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("upload cancelled before start: %w", ctx.Err())
	default:
	}

	client, err := s.getClient(encryptor)
	if err != nil {
		return err
	}

	baseKey := s.buildObjectKey(fileName)
	innerPartSize := s.innerPartSize()

	// Look ahead by one inner part to choose the layout. A stream that fits in a single inner part
	// (WAL segments, .metadata/.manifest sidecars, small dumps, empty files) is written as one
	// plain object {fileName} with no manifest, exactly as legacy backups are. Anything larger is
	// split across {fileName}.partNNNNNN objects described by a {fileName}.parts manifest.
	firstPart := make([]byte, innerPartSize)
	firstPartLen, readErr := io.ReadFull(file, firstPart)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		return fmt.Errorf("read error: %w", readErr)
	}

	if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
		return s.uploadSingleObject(ctx, client, baseKey, firstPart[:firstPartLen])
	}

	// The first part filled exactly: peek one more byte to tell "stream is exactly one inner part"
	// (still single object, e.g. a 16 MiB WAL segment) from "stream is larger" (chunked).
	peek := make([]byte, 1)
	peekLen, peekErr := io.ReadFull(file, peek)
	if peekErr != nil && peekErr != io.EOF {
		return fmt.Errorf("read error: %w", peekErr)
	}

	if peekLen == 0 {
		return s.uploadSingleObject(ctx, client, baseKey, firstPart)
	}

	coreClient, err := s.getCoreClient(encryptor)
	if err != nil {
		return err
	}

	source := io.MultiReader(bytes.NewReader(firstPart), bytes.NewReader(peek[:peekLen]), file)

	return s.uploadChunked(ctx, coreClient, client, baseKey, source)
}

func (s *S3Storage) GetFile(
	encryptor encryption.FieldEncryptor,
	fileName string,
) (io.ReadCloser, error) {
	client, err := s.getClient(encryptor)
	if err != nil {
		return nil, err
	}

	baseKey := s.buildObjectKey(fileName)

	manifest, hasManifest, err := s.readManifest(context.TODO(), client, baseKey)
	if err != nil {
		return nil, err
	}

	if hasManifest {
		return newReassemblingReader(client, s.S3Bucket, manifest.Parts), nil
	}

	object, err := client.GetObject(
		context.TODO(),
		s.S3Bucket,
		baseKey,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from S3: %w", err)
	}

	// Check if the file actually exists by reading the first byte
	buf := make([]byte, 1)
	_, readErr := object.Read(buf)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		_ = object.Close()
		return nil, fmt.Errorf("file does not exist in S3: %w", readErr)
	}

	// Reset the reader to the beginning
	_, seekErr := object.Seek(0, io.SeekStart)
	if seekErr != nil {
		_ = object.Close()
		return nil, fmt.Errorf("failed to reset file reader: %w", seekErr)
	}

	return object, nil
}

func (s *S3Storage) DeleteFile(encryptor encryption.FieldEncryptor, fileName string) error {
	client, err := s.getClient(encryptor)
	if err != nil {
		return err
	}

	baseKey := s.buildObjectKey(fileName)

	ctx, cancel := context.WithTimeout(context.Background(), s3DeleteTimeout)
	defer cancel()

	manifest, hasManifest, err := s.readManifest(ctx, client, baseKey)
	if err != nil {
		return err
	}

	if hasManifest {
		keys := make([]string, 0, len(manifest.Parts)+1)
		for _, part := range manifest.Parts {
			keys = append(keys, part.Key)
		}
		keys = append(keys, manifestObjectKey(baseKey))

		if err := s.removeObjects(ctx, client, keys); err != nil {
			return fmt.Errorf("failed to delete chunked backup from S3: %w", err)
		}

		return nil
	}

	err = client.RemoveObject(
		ctx,
		s.S3Bucket,
		baseKey,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) Validate(encryptor encryption.FieldEncryptor) error {
	if s.S3Bucket == "" {
		return errors.New("S3 bucket is required")
	}
	if s.S3AccessKey == "" {
		return errors.New("S3 access key is required")
	}
	if s.S3SecretKey == "" {
		return errors.New("S3 secret key is required")
	}

	return nil
}

func (s *S3Storage) TestConnection(encryptor encryption.FieldEncryptor) error {
	client, err := s.getClient(encryptor)
	if err != nil {
		return err
	}

	// Create a context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the bucket exists to verify connection
	exists, err := client.BucketExists(ctx, s.S3Bucket)
	if err != nil {
		// Check if the error is due to context deadline exceeded
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("failed to connect to the bucket. Please check params")
		}
		return fmt.Errorf("failed to connect to S3: %w", err)
	}

	if !exists {
		return fmt.Errorf("bucket '%s' does not exist", s.S3Bucket)
	}

	// Test write and delete permissions by uploading and removing a small test file
	testFileID := uuid.New().String() + "-test"
	testObjectKey := s.buildObjectKey(testFileID)
	testData := []byte("test connection")
	testReader := bytes.NewReader(testData)

	// Upload test file
	_, err = client.PutObject(
		ctx,
		s.S3Bucket,
		testObjectKey,
		testReader,
		int64(len(testData)),
		minio.PutObjectOptions{
			SendContentMd5: true,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to upload test file to S3: %w", err)
	}

	// Delete test file
	err = client.RemoveObject(
		ctx,
		s.S3Bucket,
		testObjectKey,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete test file from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) HideSensitiveData() {
	s.S3AccessKey = ""
	s.S3SecretKey = ""
}

func (s *S3Storage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	var err error

	if s.S3AccessKey != "" {
		s.S3AccessKey, err = encryptor.Encrypt(s.S3AccessKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt S3 access key: %w", err)
		}
	}

	if s.S3SecretKey != "" {
		s.S3SecretKey, err = encryptor.Encrypt(s.S3SecretKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt S3 secret key: %w", err)
		}
	}

	return nil
}

func (s *S3Storage) Update(incoming *S3Storage) {
	s.S3Bucket = incoming.S3Bucket
	s.S3Region = incoming.S3Region
	s.S3Endpoint = incoming.S3Endpoint
	s.S3UseVirtualHostedStyle = incoming.S3UseVirtualHostedStyle
	s.SkipTLSVerify = incoming.SkipTLSVerify
	s.S3StorageClass = incoming.S3StorageClass

	if incoming.S3AccessKey != "" {
		s.S3AccessKey = incoming.S3AccessKey
	}

	if incoming.S3SecretKey != "" {
		s.S3SecretKey = incoming.S3SecretKey
	}

	// we do not allow to change the prefix after creation,
	// otherwise we will have to transfer all the data to the new prefix
}

func (s *S3Storage) innerPartSize() int {
	if s.innerPartSizeOverride > 0 {
		return s.innerPartSizeOverride
	}

	return defaultInnerPartSize
}

func (s *S3Storage) maxPartsPerObject() int {
	if s.maxPartsOverride > 0 {
		return s.maxPartsOverride
	}

	return defaultMaxPartsPerObject
}

func (s *S3Storage) uploadSingleObject(
	ctx context.Context,
	client *minio.Client,
	objectKey string,
	data []byte,
) error {
	opts := s.putObjectOptions()
	opts.SendContentMd5 = true

	_, err := client.PutObject(ctx, s.S3Bucket, objectKey, bytes.NewReader(data), int64(len(data)), opts)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) uploadChunked(
	ctx context.Context,
	coreClient *minio.Core,
	client *minio.Client,
	baseKey string,
	source io.Reader,
) error {
	innerPartSize := s.innerPartSize()
	maxParts := s.maxPartsPerObject()
	buf := make([]byte, innerPartSize)

	var manifestParts []manifestPart
	var uploadedKeys []string
	var totalSize int64

	// abandon undoes a failed chunked upload: it aborts the in-flight multipart (when uploadID is set)
	// and deletes every already-completed part object. It uses a fresh context detached from ctx
	// because the usual reason we land here is ctx cancellation, and cleanup issued on a cancelled
	// context is a no-op — which would leave completed part objects with no manifest as dead weight in
	// the bucket until retention or an overwrite reclaims them.
	abandon := func(objectKey, uploadID string) {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), s3DeleteTimeout)
		defer cancel()

		if uploadID != "" {
			_ = coreClient.AbortMultipartUpload(cleanupCtx, s.S3Bucket, objectKey, uploadID)
		}

		if len(uploadedKeys) > 0 {
			_ = s.removeObjects(cleanupCtx, client, uploadedKeys)
		}
	}

	objectIndex := 1
	sourceEnded := false

	for !sourceEnded {
		objectKey := partObjectKey(baseKey, objectIndex)

		uploadID, err := coreClient.NewMultipartUpload(ctx, s.S3Bucket, objectKey, s.putObjectOptions())
		if err != nil {
			abandon("", "")
			return fmt.Errorf("failed to initiate multipart upload for %s: %w", objectKey, err)
		}

		var parts []minio.CompletePart
		objectHasher := sha256.New()
		var objectSize int64

		for partNumber := 1; partNumber <= maxParts; partNumber++ {
			select {
			case <-ctx.Done():
				abandon(objectKey, uploadID)
				return fmt.Errorf("upload cancelled: %w", ctx.Err())
			default:
			}

			partLen, readErr := io.ReadFull(source, buf)
			if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
				abandon(objectKey, uploadID)
				return fmt.Errorf("read error: %w", readErr)
			}

			if partLen == 0 && readErr == io.EOF {
				sourceEnded = true
				break
			}

			if err := s.putInnerPart(
				ctx,
				coreClient,
				objectKey,
				uploadID,
				partNumber,
				buf[:partLen],
				objectHasher,
				&parts,
			); err != nil {
				abandon(objectKey, uploadID)
				return err
			}

			objectSize += int64(partLen)

			if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
				sourceEnded = true
				break
			}
		}

		// The source ended exactly on the previous object's boundary, so this object got no data.
		if len(parts) == 0 {
			_ = coreClient.AbortMultipartUpload(ctx, s.S3Bucket, objectKey, uploadID)
			break
		}

		if _, err := coreClient.CompleteMultipartUpload(
			ctx,
			s.S3Bucket,
			objectKey,
			uploadID,
			parts,
			s.putObjectOptions(),
		); err != nil {
			abandon(objectKey, uploadID)
			return fmt.Errorf("failed to complete multipart upload for %s: %w", objectKey, err)
		}

		uploadedKeys = append(uploadedKeys, objectKey)
		manifestParts = append(manifestParts, manifestPart{
			Key:    objectKey,
			Size:   objectSize,
			SHA256: hex.EncodeToString(objectHasher.Sum(nil)),
		})
		totalSize += objectSize
		objectIndex++
	}

	manifest := chunkedManifest{
		Version:   manifestVersion,
		TotalSize: totalSize,
		Parts:     manifestParts,
	}

	if err := s.writeManifest(ctx, client, baseKey, manifest); err != nil {
		abandon("", "")
		return err
	}

	return nil
}

func (s *S3Storage) putInnerPart(
	ctx context.Context,
	coreClient *minio.Core,
	objectKey string,
	uploadID string,
	partNumber int,
	partData []byte,
	objectHasher hash.Hash,
	parts *[]minio.CompletePart,
) error {
	partMd5 := md5.Sum(partData)
	md5Base64 := base64.StdEncoding.EncodeToString(partMd5[:])

	part, err := coreClient.PutObjectPart(
		ctx,
		s.S3Bucket,
		objectKey,
		uploadID,
		partNumber,
		bytes.NewReader(partData),
		int64(len(partData)),
		minio.PutObjectPartOptions{Md5Base64: md5Base64},
	)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("upload cancelled: %w", ctx.Err())
		}

		return fmt.Errorf("failed to upload part %d of %s: %w", partNumber, objectKey, err)
	}

	objectHasher.Write(partData)
	*parts = append(*parts, minio.CompletePart{PartNumber: partNumber, ETag: part.ETag})

	return nil
}

func (s *S3Storage) putObjectOptions() minio.PutObjectOptions {
	return minio.PutObjectOptions{
		StorageClass: string(s.S3StorageClass),
	}
}

func (s *S3Storage) buildObjectKey(fileName string) string {
	if s.S3Prefix == "" {
		return fileName
	}

	prefix := s.S3Prefix
	prefix = strings.TrimPrefix(prefix, "/")

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return prefix + fileName
}

func (s *S3Storage) getClient(encryptor encryption.FieldEncryptor) (*minio.Client, error) {
	endpoint, useSSL, accessKey, secretKey, bucketLookup, transport, err := s.getClientParams(
		encryptor,
	)
	if err != nil {
		return nil, err
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		Region:       s.S3Region,
		BucketLookup: bucketLookup,
		Transport:    transport,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	return minioClient, nil
}

func (s *S3Storage) getCoreClient(encryptor encryption.FieldEncryptor) (*minio.Core, error) {
	endpoint, useSSL, accessKey, secretKey, bucketLookup, transport, err := s.getClientParams(
		encryptor,
	)
	if err != nil {
		return nil, err
	}

	coreClient, err := minio.NewCore(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       useSSL,
		Region:       s.S3Region,
		BucketLookup: bucketLookup,
		Transport:    transport,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO Core client: %w", err)
	}

	return coreClient, nil
}

func (s *S3Storage) getClientParams(
	encryptor encryption.FieldEncryptor,
) (endpoint string, useSSL bool, accessKey, secretKey string, bucketLookup minio.BucketLookupType, transport *http.Transport, err error) {
	endpoint = s.S3Endpoint
	useSSL = true

	if after, ok := strings.CutPrefix(endpoint, "http://"); ok {
		useSSL = false
		endpoint = after
	} else if after, ok := strings.CutPrefix(endpoint, "https://"); ok {
		endpoint = after
	}

	if endpoint == "" {
		endpoint = fmt.Sprintf("s3.%s.amazonaws.com", s.S3Region)
	}

	accessKey, err = encryptor.Decrypt(s.S3AccessKey)
	if err != nil {
		return "", false, "", "", 0, nil, fmt.Errorf("failed to decrypt S3 access key: %w", err)
	}

	secretKey, err = encryptor.Decrypt(s.S3SecretKey)
	if err != nil {
		return "", false, "", "", 0, nil, fmt.Errorf("failed to decrypt S3 secret key: %w", err)
	}

	bucketLookup = minio.BucketLookupAuto
	if s.S3UseVirtualHostedStyle {
		bucketLookup = minio.BucketLookupDNS
	}

	transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: s3ConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   s3TLSHandshakeTimeout,
		ResponseHeaderTimeout: s3ResponseTimeout,
		IdleConnTimeout:       s3IdleConnTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: s.SkipTLSVerify,
		},
	}

	return endpoint, useSSL, accessKey, secretKey, bucketLookup, transport, nil
}

// chunkedManifest lists, in order, the objects a single logical file was split into, with each
// object's size and sha256. It is written LAST, after every part object is durably committed.
//
// Why a manifest instead of just probing {fileName}.part1, .part2, ... until a key 404s:
//
//  1. The bytes we store are an encrypted stream with no end-of-stream marker (see
//     util/encryption). Probe-until-404 cannot tell a complete backup from one whose final part
//     failed to upload, was truncated, or was reaped by lifecycle rules: it would stop at the gap
//     and hand the restore a silently short, corrupt stream. TotalSize plus the per-part list and
//     sha256 let GetFile reject a missing/short/corrupted part BEFORE the restore starts, turning
//     silent data loss into a hard error.
//  2. Because the manifest is written last, its presence is the atomic "backup is complete" marker.
//     A crash mid-upload leaves part objects but no manifest, which GetFile treats as a hard error
//     (it finds neither manifest nor single object) rather than a valid N-part backup.
//  3. It immunizes reads against S3 eventual consistency: a transient 404 on a middle part can no
//     longer be misread as the end of the stream.
type chunkedManifest struct {
	Version   int            `json:"version"`
	TotalSize int64          `json:"totalSize"`
	Parts     []manifestPart `json:"parts"`
}

type manifestPart struct {
	Key    string `json:"key"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func partObjectKey(baseKey string, objectIndex int) string {
	return fmt.Sprintf("%s.part%06d", baseKey, objectIndex)
}

func manifestObjectKey(baseKey string) string {
	return baseKey + manifestSuffix
}

func (s *S3Storage) writeManifest(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
	manifest chunkedManifest,
) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk manifest: %w", err)
	}

	opts := s.putObjectOptions()
	opts.SendContentMd5 = true
	opts.ContentType = "application/json"

	_, err = client.PutObject(
		ctx,
		s.S3Bucket,
		manifestObjectKey(baseKey),
		bytes.NewReader(body),
		int64(len(body)),
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to write chunk manifest: %w", err)
	}

	return nil
}

func (s *S3Storage) readManifest(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
) (chunkedManifest, bool, error) {
	var manifest chunkedManifest

	manifestKey := manifestObjectKey(baseKey)

	_, err := client.StatObject(ctx, s.S3Bucket, manifestKey, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return manifest, false, nil
		}

		return manifest, false, fmt.Errorf("failed to stat chunk manifest: %w", err)
	}

	object, err := client.GetObject(ctx, s.S3Bucket, manifestKey, minio.GetObjectOptions{})
	if err != nil {
		return manifest, false, fmt.Errorf("failed to read chunk manifest: %w", err)
	}
	defer func() { _ = object.Close() }()

	body, err := io.ReadAll(object)
	if err != nil {
		return manifest, false, fmt.Errorf("failed to read chunk manifest: %w", err)
	}

	if err := json.Unmarshal(body, &manifest); err != nil {
		return manifest, false, fmt.Errorf("failed to parse chunk manifest: %w", err)
	}

	return manifest, true, nil
}

func (s *S3Storage) removeObjects(ctx context.Context, client *minio.Client, keys []string) error {
	objectsCh := make(chan minio.ObjectInfo, len(keys))
	for _, key := range keys {
		objectsCh <- minio.ObjectInfo{Key: key}
	}
	close(objectsCh)

	var firstErr error
	for removeResult := range client.RemoveObjects(ctx, s.S3Bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if removeResult.Err != nil && firstErr == nil {
			firstErr = removeResult.Err
		}
	}

	return firstErr
}

// reassemblingReader streams a chunked backup back as one byte stream by opening each part object
// in manifest order, reading it to EOF and only then opening the next. It holds at most one part
// connection at a time, preserving the bounded-memory streaming of the upload path (ADR-0004), and
// verifies each part's length and sha256 against the manifest so a missing, truncated, or corrupted
// part surfaces as a read error instead of a silently short stream.
type reassemblingReader struct {
	client  *minio.Client
	bucket  string
	parts   []manifestPart
	index   int
	current *minio.Object
	hasher  hash.Hash
	read    int64
}

func newReassemblingReader(client *minio.Client, bucket string, parts []manifestPart) *reassemblingReader {
	return &reassemblingReader{client: client, bucket: bucket, parts: parts}
}

func (r *reassemblingReader) Read(p []byte) (int, error) {
	for {
		if r.current == nil {
			if r.index >= len(r.parts) {
				return 0, io.EOF
			}

			object, err := r.client.GetObject(context.TODO(), r.bucket, r.parts[r.index].Key, minio.GetObjectOptions{})
			if err != nil {
				return 0, fmt.Errorf("failed to open chunk %s: %w", r.parts[r.index].Key, err)
			}

			r.current = object
			r.hasher = sha256.New()
			r.read = 0
		}

		n, err := r.current.Read(p)
		if n > 0 {
			r.hasher.Write(p[:n])
			r.read += int64(n)
		}

		if errors.Is(err, io.EOF) {
			if verifyErr := r.finishCurrentPart(); verifyErr != nil {
				return n, verifyErr
			}

			if n > 0 {
				return n, nil
			}

			continue
		}

		if err != nil {
			return n, fmt.Errorf("failed to read chunk %s: %w", r.parts[r.index].Key, err)
		}

		return n, nil
	}
}

func (r *reassemblingReader) Close() error {
	if r.current == nil {
		return nil
	}

	err := r.current.Close()
	r.current = nil

	return err
}

func (r *reassemblingReader) finishCurrentPart() error {
	part := r.parts[r.index]

	_ = r.current.Close()
	r.current = nil

	if r.read != part.Size {
		return fmt.Errorf("chunk %s size mismatch: manifest %d, read %d", part.Key, part.Size, r.read)
	}

	if checksum := hex.EncodeToString(r.hasher.Sum(nil)); checksum != part.SHA256 {
		return fmt.Errorf("chunk %s checksum mismatch", part.Key)
	}

	r.index++

	return nil
}
