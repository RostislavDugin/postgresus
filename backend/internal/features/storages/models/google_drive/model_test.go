package google_drive_storage

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	test_utils "databasus-backend/internal/util/testing"
)

func Test_BuildHTTPClient_Transport_ResolvesProxyFromEnvironment(t *testing.T) {
	transport := getGoogleDriveBaseTransport(t)

	require.NotNil(t, transport.Proxy)
	assert.Equal(t,
		reflect.ValueOf(http.ProxyFromEnvironment).Pointer(),
		reflect.ValueOf(transport.Proxy).Pointer(),
	)
}

func Test_BuildHTTPClient_TransportThroughProxy_TunnelsAndRejectsUntrustedCertificate(t *testing.T) {
	exchange := test_utils.SendThroughTunnelProxy(t, getGoogleDriveBaseTransport(t), "Bearer access-token")

	test_utils.AssertProxyReceivedOnlyTunnelRequest(t, exchange)
	test_utils.AssertCertificateRejected(t, exchange)
}

func getGoogleDriveBaseTransport(t *testing.T) *http.Transport {
	t.Helper()

	client := (&GoogleDriveStorage{}).buildHTTPClient(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "access-token"}),
	)

	oauthTransport, isOAuthTransport := client.Transport.(*oauth2.Transport)
	require.True(t, isOAuthTransport, "Google Drive client must authenticate through oauth2.Transport")

	baseTransport, isTransport := oauthTransport.Base.(*http.Transport)
	require.True(t, isTransport, "Google Drive OAuth transport must sit on an *http.Transport")

	return baseTransport
}
