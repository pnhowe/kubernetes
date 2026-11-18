package contractor

import (
	"context"
	"testing"
	"time"
)

func TestGetClientNotSetup(t *testing.T) {
	factory = nil
	if _, err := GetClient(context.Background()); err == nil {
		t.Fatal("expected an error when the factory is not setup")
	}
}

func TestSetupFactoryNeedsHost(t *testing.T) {
	if err := SetupFactory("", "user", "pass", ""); err == nil {
		t.Fatal("expected an error for a blank hostname")
	}
}

// contractor being down must not stop the operator from starting, or panic when it is first used,
// it is an error each time the client is asked for, so it can recover once contractor is up
func TestGetClientContractorDown(t *testing.T) {
	// nothing is listening on port 1
	if err := SetupFactory("http://127.0.0.1:1", "user", "pass", ""); err != nil {
		t.Fatalf("setup should not connect: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := 0; i < 2; i++ {
		client, err := GetClient(ctx)
		if err == nil || client != nil {
			t.Fatalf("attempt %d: expected an error and no client, got client=%v err=%v", i, client, err)
		}
	}

	if factory.client != nil {
		t.Fatal("a failed connection must not be remembered as a client")
	}
}
