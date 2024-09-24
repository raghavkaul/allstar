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
	"sort"
	"testing"
)

func TestListUpstreamBranches(t *testing.T) {
	tests := []struct {
		repo     string
		expected []string
	}{
		{
			repo:     "https://github.com/ossf/allstar",
			expected: []string{"main", "prod-temp"},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()
		lsc, err := GetLocal(ctx, tt.repo)

		if err != nil {
			t.Errorf("GetLocal: %+v", err)
		}

		branches, err := lsc.ListUpstreamBranches()

		if err != nil {
			t.Errorf("ListUpstreamBranches: %+v", err)
		}

		if !slicesEqUnordered(branches, tt.expected) {
			t.Errorf("wrong upstream branches, expected: %+v actual %+v", tt.expected, branches)
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
