// Copyright 2021 Allstar Authors

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ghclients stores ghclients with caching and auth for installations
// of a GitHub App
package ghclients

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v59/github"
	"github.com/gregjones/httpcache"
	"github.com/ossf/allstar/pkg/config/operator"
	"gocloud.dev/runtimevar"
	_ "gocloud.dev/runtimevar/awssecretsmanager"
	_ "gocloud.dev/runtimevar/filevar"
	_ "gocloud.dev/runtimevar/gcpsecretmanager"
)

var ghinstallationNewAppsTransport func(http.RoundTripper, int64,
	[]byte) (*ghinstallation.AppsTransport, error)
var ghinstallationNew func(http.RoundTripper, int64, int64, []byte) (
	*ghinstallation.Transport, error)
var getKey func(context.Context) ([]byte, error)
var getKeyFromSecret func(context.Context, string) ([]byte, error)

var privateKey = operator.PrivateKey
var keySecret = operator.KeySecret

func init() {
	ghinstallationNewAppsTransport = ghinstallation.NewAppsTransport
	ghinstallationNew = ghinstallation.New
	getKey = getKeyReal
	getKeyFromSecret = getKeyFromSecretReal
}

type GhClientsInterface interface {
	Get(i int64) (*github.Client, error)
	Free(i int64)
}

// GHClients stores clients per-installation for re-use throughout a process.
type GHClients struct {
	clients map[int64]*github.Client
	tr      http.RoundTripper
	key     []byte
}

// NewGHClients returns a new GHClients. The provided RoundTripper will be
// stored and used when creating new clients.
func NewGHClients(ctx context.Context, t http.RoundTripper) (*GHClients, error) {
	key, err := getKey(ctx)
	if err != nil {
		return nil, err
	}
	return &GHClients{
		clients: make(map[int64]*github.Client),
		tr:      t,
		key:     key,
	}, nil
}

func (g *GHClients) Free(i int64) {
	delete(g.clients, i)
}

// Get gets the client for installation id i, If i is 0 it gets the client for
// the app-level api. If a stored client is not available, it creates a new
// client with auth and caching built in.
func (g *GHClients) Get(i int64) (*github.Client, error) {
	if c, ok := g.clients[i]; ok {
		return c, nil
	}

	ctr := &httpcache.Transport{
		Transport:           g.tr,
		Cache:               newMemoryCache(),
		MarkCachedResponses: true,
	}

	var tr http.RoundTripper
	var err error
	if i == 0 {
		tr, err = ghinstallationNewAppsTransport(ctr, operator.AppID, g.key)
	} else {
		tr, err = ghinstallationNew(ctr, operator.AppID, i, g.key)
	}
	if err != nil {
		return nil, err
	}
	g.clients[i] = github.NewClient(&http.Client{Transport: tr})
	return g.clients[i], nil
}

func getKeyFromSecretReal(ctx context.Context, keySecretVal string) ([]byte, error) {
	v, err := runtimevar.OpenVariable(ctx, keySecretVal)
	if err != nil {
		return nil, err
	}
	defer v.Close()
	s, err := v.Latest(ctx)
	if err != nil {
		return nil, err
	}
	return s.Value.([]byte), nil
}

func getKeyReal(ctx context.Context) ([]byte, error) {
	if keySecret == "direct" {
		return []byte(privateKey), nil
	}
	return getKeyFromSecret(ctx, keySecret)
}

// Get GitHub App installation access token.
func GetInstallationAccessToken(ctx context.Context, c *github.Client, org string) (string, error) {
	inst, resp, err := c.Apps.FindOrganizationInstallation(ctx, org)

	if err != nil || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("couldn't find GH app installation for org %s: %w (%s)", org, err, resp.Status)
	}

	url, err := url.Parse(inst.GetAccessTokensURL())
	if err != nil {
		return "", fmt.Errorf("url.Parse: %s %w", url, err)
	}

	var respJson struct {
		Token string `json:"token"`
	}

	// github.Client should set Authorization header with JWT
	resp, err = c.Do(ctx, &http.Request{
		Method: "POST",
		URL:    url,
	}, respJson)

	if err != nil || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST %s: %s %w", url, resp.Status, err)
	}

	return respJson.Token, nil
}
