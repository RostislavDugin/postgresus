package s3_storage

import "errors"

// These are terminal on the read path: retrying re-reads the same wrong bytes, and passing them
// downstream would corrupt a stream the decryptor cannot resynchronise (see chunkedManifest).
var (
	ErrChunkSizeMismatch     = errors.New("s3 chunk size mismatch")
	ErrChunkChecksumMismatch = errors.New("s3 chunk checksum mismatch")

	// ErrRangeNotHonoured means a resumed GET was answered with a different byte span than the
	// Range header asked for, which some S3-compatible backends do by replying 200 with the whole
	// object. Resuming from byte 0 mid-stream is what this guards against.
	ErrRangeNotHonoured = errors.New("s3 range request not honoured")

	ErrReadRetryBudgetExhausted = errors.New("s3 chunk read retries exhausted")

	ErrReaderClosed = errors.New("s3 chunk reader is closed")
)
