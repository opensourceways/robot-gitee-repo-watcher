package main

import (
	"github.com/panjf2000/ants/v2"
	"sync"

	sdk "gitee.com/openeuler/go-gitee/gitee"
)

const botName = "repo-watcher"

type iClient interface {
	GetRepos(org string) ([]sdk.Project, error)
	GetPathContent(org, repo, path, ref string) (sdk.Content, error)
	GetDirectoryTree(org, repo, sha string, recursive int32) (sdk.Tree, error)
	GetRepoAllBranch(org, repo string) ([]sdk.Branch, error)
	GetGiteeRepo(org, repo string) (sdk.Project, error)
	CreateBranch(org, repo, branch, parentBranch string) error
	SetProtectionBranch(org, repo, branch string) error
	CancelProtectionBranch(org, repo, branch string) error
	RemoveRepoMember(org, repo, login string) error
	AddRepoMember(org, repo, login, permission string) error
	CreateRepo(org string, repo sdk.RepositoryPostParam) error
	SetRepoReviewer(org, repo string, reviewer sdk.SetRepoReviewer) error
	UpdateRepo(org, repo string, info sdk.RepoPatchParam) error
}

func newRobot(cli iClient, pool *ants.Pool) *robot {
	return &robot{cli: cli, pool: pool}
}

type robot struct {
	pool *ants.Pool
	cli  iClient
	wg   sync.WaitGroup
}
