package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	stosapi "github.com/storageos/go-api/autogenerated"
	stosapierrors "github.com/storageos/go-api/errors"
)

const (
	// secretUsernameKey is the key in the secret that holds the username value.
	secretUsernameKey = "username"
	// secretPasswordKey is the key in the secret that holds the password value.
	secretPasswordKey = "password"
	// DefaultPort is the default api port.
	DefaultPort = 5705
	// DefaultScheme is used for api endpoint.
	DefaultScheme = "http"
	// TLSScheme scheme can be used if the api endpoint has TLS enabled.
	TLSScheme = "https"
)

var (
	// HTTPTimeout is the time limit for requests made by the API Client. The
	// timeout includes connection time, any redirects, and reading the response
	// body. The timer remains running after Get, Head, Post, or Do return and
	// will interrupt reading of the Response.Body.
	HTTPTimeout = 10 * time.Second
	// AuthenticationTimeout is the time limit for authentication requests to
	// complete.  It should be longer than the HTTPTimeout.
	AuthenticationTimeout = 20 * time.Second
	// ErrNoAuthToken is returned when the API client did not get an error
	// during authentication but no valid auth token was returned.
	ErrNoAuthToken = errors.New("no token found in auth response")
)

type VolumePVC struct {
	ID  string
	PVC string
}

type ControlPlane interface {
	RefreshJwt(ctx context.Context) (stosapi.UserSession, *http.Response, error)
	AuthenticateUser(ctx context.Context, authUserData stosapi.AuthUserData) (stosapi.UserSession, *http.Response, error)
	ListNamespaces(ctx context.Context) ([]stosapi.Namespace, *http.Response, error)
	ListVolumes(ctx context.Context, namespaceID string) ([]stosapi.Volume, *http.Response, error)
}

// Client provides access to the StorageOS API.
type Client struct {
	ctx context.Context
	api ControlPlane
}

// NewOndatHttpClient returns a pre-authenticated client for the StorageOS API. The
// authentication token must be refreshed periodically using AuthenticateRefresh().
func NewOndatHttpClient(username, password, endpoint string) (*Client, error) {
	transport := http.DefaultTransport
	ctx, client, err := newAuthenticatedClient(username, password, endpoint, transport)
	if err != nil {
		return nil, err
	}
	return &Client{api: client.DefaultApi, ctx: ctx}, nil
}

func newAuthenticatedClient(username, password, endpoint string, transport http.RoundTripper) (context.Context, *stosapi.APIClient, error) {
	config := stosapi.NewConfiguration()

	if !strings.Contains(endpoint, "://") {
		endpoint = fmt.Sprintf("%s://%s", DefaultScheme, endpoint)
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, err
	}

	config.Scheme = u.Scheme
	config.Host = u.Host
	if !strings.Contains(u.Host, ":") {
		config.Host = fmt.Sprintf("%s:%d", u.Host, DefaultPort)
	}

	httpc := &http.Client{
		Timeout:   HTTPTimeout,
		Transport: transport,
	}
	config.HTTPClient = httpc

	// Get a wrappered API client.
	client := stosapi.NewAPIClient(config)

	// Authenticate and return context with credentials and client.
	ctx, err := Authenticate(client, username, password)
	if err != nil {
		return nil, nil, err
	}

	return ctx, client, nil
}

// Authenticate against the API and set the authentication token in the client
// to be used for subsequent API requests.  The token must be refreshed
// periodically using AuthenticateRefresh().
func Authenticate(client *stosapi.APIClient, username, password string) (context.Context, error) {
	// Create context just for the login.
	ctx, cancel := context.WithTimeout(context.Background(), AuthenticationTimeout)
	defer cancel()

	// Initial basic auth to retrieve the jwt token.
	_, resp, err := client.DefaultApi.AuthenticateUser(ctx, stosapi.AuthUserData{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, stosapierrors.MapAPIError(err, resp)
	}
	defer resp.Body.Close()

	// Set auth token in a new context for re-use.
	token := respAuthToken(resp)
	if token == "" {
		return nil, ErrNoAuthToken
	}
	return context.WithValue(context.Background(), stosapi.ContextAccessToken, token), nil
}

// respAuthToken is a helper to pull the auth token out of a HTTP Response.
func respAuthToken(resp *http.Response) string {
	if value := resp.Header.Get("Authorization"); value != "" {
		// "Bearer aaaabbbbcccdddeeeff"
		return strings.Split(value, " ")[1]
	}
	return ""
}

// ReadCredsFromMountedSecret reads the api username and password from a
// Kubernetes secret mounted at the given path.  If the username or password in
// the secret changes, the data in the mounted file will also change.
func ReadCredsFromMountedSecret(path string) (string, string, error) {
	username, err := Read(filepath.Join(path, secretUsernameKey))
	if err != nil {
		return "", "", err
	}
	password, err := Read(filepath.Join(path, secretPasswordKey))
	if err != nil {
		return "", "", err
	}
	return username, password, nil
}

// Read a secret from the given path.  The secret is expected to be mounted into
// the container by Kubernetes.
func Read(path string) (string, error) {
	secretBytes, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return "", fmt.Errorf("unable to read secret: %s, error: %s", path, readErr)
	}
	val := strings.TrimSpace(string(secretBytes))
	return val, nil
}

// GetAllOndatVolumes wraps all the necessary steps to request the Ondat volumes from
// the storageos instance on this node (authentication, get ns, get vols)
func GetAllOndatVolumes(log logr.Logger, apiSecretsPath string) ([]VolumePVC, error) {
	username, password, err := ReadCredsFromMountedSecret(apiSecretsPath)
	if err != nil {
		log.Error(err, "failed to read creds")
		return nil, err
	}

	c, err := NewOndatHttpClient(username, password, "storageos")
	if err != nil {
		log.Error(err, "failed to create api client")
		return nil, err
	}

	namespaces, _, err := c.api.ListNamespaces(c.ctx)
	if err != nil {
		log.Error(err, "failed to list namespaces")
		return nil, err
	}

	res := []VolumePVC{}
	for _, ns := range namespaces {
		vols, _, err := c.api.ListVolumes(c.ctx, ns.Id)
		if err != nil {
			log.Error(err, "failed to get volumes in namespace "+ns.Name)
			continue
		}

		for _, vol := range vols {
			pvc := vol.Labels["csi.storage.k8s.io/pvc/name"]
			res = append(res, VolumePVC{ID: vol.Id, PVC: pvc})
		}
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("no Ondat volumes found")
	}
	return res, nil
}
