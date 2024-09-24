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

// Package scorecard handles sharing a Scorecard githubrepo client across
// multiple Allstar policies. The Allstar policy interface does not handle
// passing a reference to this, so we will store a master state here and look
// it up each time. We don't want to keep the tarball around forever, or we
// will run out of disk space. This is intended to be setup once for a repo,
// then all policies run, then closed for that repo.
package local

import (
	"context"
	"os"
	"strings"
	"sync"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/ossf/scorecard/v5/clients"
	"github.com/ossf/scorecard/v5/clients/localdir"
	"github.com/ossf/scorecard/v5/log"
)

// Type LocalScClient is returned from Get. It contains the clients needed to call
// scorecard checks on a local repo
type LocalScClient struct {
	localGitRepo      *git.Repository
	localWorktree     *git.Worktree
	LocalScRepo       clients.Repo
	LocalScRepoClient clients.RepoClient
}

var localScClients map[string]*LocalScClient = make(map[string]*LocalScClient)
var mMutex sync.RWMutex

const defaultGitRef = "HEAD"

var localrepoMakeLocalRepo func(string) (clients.Repo, error)
var localrepoCreateLocalDirClient func(context.Context, *log.Logger) clients.RepoClient

func init() {
	localrepoMakeLocalRepo = localdir.MakeLocalDirRepo
	localrepoCreateLocalDirClient = localdir.CreateLocalDirClient
}

// Function Get will get the scorecard clients and create them if they don't
// exist. The github repo is initialized, which means the tarball is
// downloaded.
func GetLocal(ctx context.Context, fullRepo string) (*LocalScClient, error) {
	mMutex.Lock()
	if scc, ok := localScClients[fullRepo]; ok {
		mMutex.Unlock()
		return scc, nil
	}
	scc, err := createLocal(ctx, fullRepo)
	if err != nil {
		mMutex.Unlock()
		return nil, err
	}
	localScClients[fullRepo] = scc
	mMutex.Unlock()
	return scc, nil
}

func createLocal(ctx context.Context, fullRepo string) (*LocalScClient, error) {
	ptn := strings.ReplaceAll(fullRepo, "/", "-")
	path, err := os.MkdirTemp("/tmp", ptn)
	if err != nil {
		return nil, err
	}
	// TODO: consider narrowing this down to just .github/workflow for Dangerous-Workflow policy?
	r, err := git.PlainClone(path, false, &git.CloneOptions{
		URL: fullRepo,
		// Progress: os.Stdout,
	})
	if err != nil {
		return nil, err
	}
	// Fetch all refs
	err = r.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"+refs/*:refs/*"},
	})
	if err != nil {
		return nil, err
	}
	wt, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	scr, err := localrepoMakeLocalRepo(path)
	if err != nil {
		return nil, err
	}
	l := log.NewLogger(log.InfoLevel)
	lrc := localrepoCreateLocalDirClient(ctx, l)
	if err := lrc.InitRepo(scr, defaultGitRef, 0); err != nil {
		return nil, err
	}
	return &LocalScClient{
		localGitRepo:      r,
		localWorktree:     wt,
		LocalScRepo:       scr,
		LocalScRepoClient: lrc,
	}, nil
}

func (lsc LocalScClient) ListUpstreamBranches() ([]string, error) {
	ri, err := lsc.localGitRepo.Branches()
	if err != nil {
		return nil, err
	}

	branches := []string{}
	ri.ForEach(func(r *plumbing.Reference) error {
		if !strings.HasPrefix(r.Name().String(), "refs/heads/") {
			return nil
		}
		branches = append(branches, r.Name().Short())
		return nil
	})

	return branches, nil
}

// Check out a branch on the local worktree for the local client
func (lsc LocalScClient) Checkout(ref string) error {
	opts := git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(ref),
	}
	err := lsc.localWorktree.Checkout(&opts)
	if err != nil {
		return err
	}
	return nil
}

// Function Close will close the scorecard clients. This cleans up the
// downloaded tarball.
func Close(fullRepo string) {
	mMutex.Lock()
	lc, ok := localScClients[fullRepo]
	if !ok {
		mMutex.Unlock()
		return
	}
	lc.LocalScRepoClient.Close()
	delete(localScClients, fullRepo)
	mMutex.Unlock()
}
