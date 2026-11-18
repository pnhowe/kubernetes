package contractor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	cinp "github.com/cinp/go/v2"
	"github.com/go-logr/logr"
	contractorClient "github.com/t3kton/contractor_goclient"
	ctrl "sigs.k8s.io/controller-runtime"
)

const tokenLifeTime = time.Minute * 10

// clientFactory creates authencated Contractor Clients
type clientFactory struct {
	mu           sync.Mutex // the controller and webhooks share the client, and token refresh must not race
	hostname     string
	proxy        string
	username     string
	password     string
	client       *contractorClient.Contractor
	tokenExpires time.Time
}

var factory *clientFactory = nil

// SetupFactory sets up the connection and authencation information.
// It does not connect, that happens on the first GetClient, so the operator can start (and report
// the problem in the Structure's status) when contractor is not up yet.
func SetupFactory(hostname string, username string, password string, proxy string) error {
	if hostname == "" {
		return errors.New("contractor hostname not specified")
	}

	factory = &clientFactory{hostname: hostname, proxy: proxy, username: username, password: password}

	return nil
}

// CleanupFactory cleans up the factory, logingout/cleaning up the auth token
func CleanupFactory(ctx context.Context) {
	if factory == nil {
		return
	}

	factory.mu.Lock()
	defer factory.mu.Unlock()
	if factory.client == nil {
		return
	}

	factory.client.Logout(ctx)
}

// GetClient returns a authencated Contractor client, connecting if it has not yet,
// and refreshing the token if it has expired.
// An error is returned if the factory is not setup or the connection/login fails, so callers can retry later,
// the contractor being down for a bit should not take down the controller.
func GetClient(ctx context.Context) (*contractorClient.Contractor, error) {
	if factory == nil {
		return nil, errors.New("contractor client factory not setup")
	}

	factory.mu.Lock()
	defer factory.mu.Unlock()

	if factory.client == nil {
		log := ctrl.Log.WithName("contractor")
		logger := slog.New(logr.ToSlogHandler(log))
		client, err := contractorClient.NewContractor(ctx, logger, factory.hostname, factory.proxy,
			factory.username, factory.password)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to contractor at '%s': %w", factory.hostname, err)
		}
		factory.client = client
		factory.tokenExpires = time.Now().Add(tokenLifeTime)
		return factory.client, nil
	}

	if time.Now().After(factory.tokenExpires) {
		factory.client.Logout(ctx)
		err := factory.client.Login(ctx, factory.username, factory.password)
		if err != nil {
			return nil, fmt.Errorf("unable to authenticate to contractor: %w", err)
		}
		factory.tokenExpires = time.Now().Add(tokenLifeTime)
	}

	return factory.client, nil
}

// SetupTestingFactory sets up the factory for testing
func SetupTestingFactory(ctx context.Context, cinp cinp.CInPClient) error {
	client := &contractorClient.Contractor{}
	client.OverrideCINPClient(cinp)

	factory = &clientFactory{username: "", password: ""}
	factory.client = client
	factory.tokenExpires = time.Now().Add(time.Hour * 24) // no set of tests should take longer than a day, right?

	return nil
}
