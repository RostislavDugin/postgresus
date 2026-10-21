package s3_storage

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
)

// Go keeps two idle connections per host by default, which drops connections as soon as more than
// two writes to one bucket run at once.
const s3MaxIdleConnsPerHost = 16

// Clients are still built per call from freshly decrypted credentials. Only the connections are
// shared: S3 signs every request, so a changed key takes effect on the next request even over a
// connection opened before the change, and a changed endpoint lands in its own connection pool.
var sharedTransports = map[bool]*http.Transport{
	false: newSharedTransport(false),
	true:  newSharedTransport(true),
}

func getSharedTransport(isSkipTLSVerify bool) *http.Transport {
	return sharedTransports[isSkipTLSVerify]
}

func newSharedTransport(isSkipTLSVerify bool) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: s3ConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   s3TLSHandshakeTimeout,
		ResponseHeaderTimeout: s3ResponseTimeout,
		IdleConnTimeout:       s3IdleConnTimeout,
		MaxIdleConnsPerHost:   s3MaxIdleConnsPerHost,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: isSkipTLSVerify,
		},
	}
}

type bucketRegionKey struct {
	endpoint string
	bucket   string
}

// A client built per call loses minio's own bucket-location cache, so without this every write to a
// storage with no region set would pay an extra location request.
var (
	bucketRegionsMutex sync.RWMutex
	bucketRegions      = map[bucketRegionKey]string{}
)

// A failed lookup is not cached and yields an empty region, so the client falls back to minio's own
// lookup and reports the real error on the request it was built for.
func getCachedBucketRegion(ctx context.Context, params clientParams, bucket string) string {
	key := bucketRegionKey{endpoint: params.endpoint, bucket: bucket}

	bucketRegionsMutex.RLock()
	cachedRegion, isCached := bucketRegions[key]
	bucketRegionsMutex.RUnlock()

	if isCached {
		return cachedRegion
	}

	lookupClient, err := newMinioClient(params, "")
	if err != nil {
		return ""
	}

	region, err := lookupClient.GetBucketLocation(ctx, bucket)
	if err != nil {
		return ""
	}

	bucketRegionsMutex.Lock()
	bucketRegions[key] = region
	bucketRegionsMutex.Unlock()

	return region
}
