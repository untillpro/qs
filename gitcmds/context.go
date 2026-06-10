/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import "strings"

// LoadRepoContext gathers repository state needed by Dev() in minimal git round-trips.
// Phase 1 runs five independent local git commands concurrently.
// Phase 2 issues one gh API call that requires Org/Repo obtained in phase 1.
func LoadRepoContext(wd string) (*RepoContext, error) {
	rc := &RepoContext{}

	type result struct {
		setFn func()
		err   error
	}

	const numConcurrent = 5
	ch := make(chan result, numConcurrent)

	go func() {
		v, err := HasRemote(wd, "upstream")
		ch <- result{setFn: func() { rc.UpstreamExists = v }, err: err}
	}()

	go func() {
		repo, org, err := GetRepoAndOrgName(wd)
		ch <- result{setFn: func() { rc.Repo = repo; rc.Org = org }, err: err}
	}()

	go func() {
		v, err := GetCurrentBranchName(wd)
		ch <- result{setFn: func() { rc.CurrentBranch = v }, err: err}
	}()

	go func() {
		v, err := GetMainBranch(wd)
		ch <- result{setFn: func() { rc.MainBranch = v }, err: err}
	}()

	go func() {
		v, err := HaveUncommittedChanges(wd)
		ch <- result{setFn: func() { rc.HasUncommittedChanges = v }, err: err}
	}()

	for range numConcurrent {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		r.setFn()
	}

	rc.IsMain = strings.EqualFold(rc.CurrentBranch, rc.MainBranch)

	// Phase 2: gh API call needs Org/Repo from phase 1
	parentRepo, err := getParentRepoByOrgRepo(wd, rc.Org, rc.Repo)
	if err != nil {
		return nil, err
	}
	rc.ParentRepo = parentRepo

	return rc, nil
}
