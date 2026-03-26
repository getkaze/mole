package github

import (
	"fmt"
	"net/http"
	"sync"

	ghinstall "github.com/bradleyfalzon/ghinstallation/v2"
	gh "github.com/google/go-github/v72/github"
)

type ClientFactory struct {
	appID          int64
	privateKeyPath string

	mu      sync.Mutex
	clients map[int64]*gh.Client
}

func NewClientFactory(appID int64, privateKeyPath string) *ClientFactory {
	return &ClientFactory{
		appID:          appID,
		privateKeyPath: privateKeyPath,
		clients:        make(map[int64]*gh.Client),
	}
}

func (f *ClientFactory) Client(installationID int64) (*gh.Client, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if c, ok := f.clients[installationID]; ok {
		return c, nil
	}

	transport, err := ghinstall.NewKeyFromFile(
		http.DefaultTransport,
		f.appID,
		installationID,
		f.privateKeyPath,
	)
	if err != nil {
		return nil, fmt.Errorf("creating github transport: %w", err)
	}

	client := gh.NewClient(&http.Client{Transport: transport})
	f.clients[installationID] = client
	return client, nil
}
