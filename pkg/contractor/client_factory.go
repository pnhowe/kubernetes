package contractor

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-logr/logr"
	contractor "github.com/t3kton/contractor_goclient"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const tokenLifeTime = time.Minute * 10

// clientFactory creates authencated Contractor Clients
type clientFactory struct {
	username     string
	password     string
	client       *contractor.Contractor
	tokenExpires time.Time
}

var factory *clientFactory = nil

// SetupFactory sets up and checks the connection and authencation information
func SetupFactory(ctx context.Context, hostname string, username string, password string, proxy string) error {
	log := log.FromContext(ctx).WithName("contractor")
	sloger := slog.New(logr.ToSlogHandler(log))

	client, err := contractor.NewContractor(ctx, sloger, hostname, proxy, username, password)
	if err != nil {
		return err
	}

	factory = &clientFactory{username: username, password: password}
	factory.client = client
	factory.tokenExpires = time.Now().Add(tokenLifeTime)

	return nil
}

// CleanupFactory cleans up the factory, logingout/cleaning up the auth token
func CleanupFactory(ctx context.Context) {
	if factory == nil || factory.client == nil {
		return
	}

	factory.client.Logout(ctx)
}

// GetClient returns a authencated Contractor client
func GetClient(ctx context.Context) *contractor.Contractor {
	if factory == nil || factory.client == nil {
		panic("Contractor Client Factory Not Setup")
	}
	if time.Now().Compare(factory.tokenExpires) == 1 {
		factory.client.Logout(ctx)
		err := factory.client.Login(ctx, factory.username, factory.password)
		if err != nil {
			panic("Unable to Authencated to Contractor, creds no longer work")
		}
		factory.tokenExpires = time.Now().Add(tokenLifeTime)
	}

	return factory.client
}

// SetupTestingFactory sets up the factory for testing
func SetupTestingFactory(ctx context.Context) error {
	client := &contractor.Contractor{}

	factory = &clientFactory{username: "", password: ""}
	factory.client = client
	factory.tokenExpires = time.Now().Add(time.Hour * 24) // no set of tests should take longer than a day, right?

	return nil
}
