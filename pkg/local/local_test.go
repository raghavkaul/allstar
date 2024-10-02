// Copyright 2024 Allstar Authors

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package local

import (
	"context"
	"net/http"
	"sort"
	"testing"

	"github.com/ossf/allstar/pkg/ghclients"
)

func Test_LocalCheckout(t *testing.T) {
	tests := []struct {
		desc     string
		repo     string
		expected []string
	}{
		{
			desc:     "basic repo",
			repo:     "google-test/.allstar",
			expected: []string{"master"},
		},
	}

	ctx := context.Background()
	ghc, err := ghclients.NewGHClients(ctx, http.DefaultTransport)
	if err != nil {
		t.Errorf("NewGHClients: %+v", err)
	}

	// TODO: Test all InstallationIDs
	gc, err := ghc.Get(0)
	if err != nil {
		t.Errorf("Get %d: %+v", 0, err)
	}

	// rlo := github.OrganizationsListOptions{}
	// repos, _, _ := gc.Organizations.ListAll(ctx, &rlo)
	// fmt.Printf("reached mid: %+v\n", repos)

	for _, tt := range tests {
		t.Logf("%s", tt.desc)
		lsc, err := GetLocal(ctx, gc, tt.repo)
		if lsc == nil || err != nil {
			t.Fatalf("GetLocal: %+v", err)
		}

		branches, err := lsc.ListUpstreamBranches()
		if err != nil {
			t.Errorf("ListUpstreamBranches: %+v", err)
		}

		if !slicesEqUnordered(branches, tt.expected) {
			t.Errorf("wrong upstream branches, expected: %+v actual %+v", tt.expected, branches)
		}

		for _, b := range branches {
			err = lsc.Checkout(b)
			if err != nil {
				t.Errorf("Checkout %s: %+v", b, err)
			}
		}

	}
}

func slicesEqUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for _, val := range a {
		i := sort.SearchStrings(b, val)
		if i >= len(val) || b[i] != val {
			return false
		}
	}

	return true
}
