package main

import (
	"strconv"

	sdk "gitee.com/openeuler/go-gitee/gitee"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
	"github.com/opensourceways/robot-gitee-repo-watcher/models"
)

func (bot *robot) createRepo(expectRepo *expectRepoInfo) models.RepoState {
	org := expectRepo.org
	repo := expectRepo.expectRepoState
	repoName := repo.Name

	if repo.RenameFrom != "" && repo.RenameFrom != repoName {
		return bot.renameRepo(org, repo)
	}

	err := bot.cli.CreateRepo(org, sdk.RepositoryPostParam{
		Name:        repoName,
		Description: repo.Description,
		HasIssues:   true,
		HasWiki:     true,
		AutoInit:    true, // set `auto_init` as true to initialize `master` branch with README after repo creation
		CanComment:  repo.Commentable,
		Private:     repo.IsPrivate(),
	})
	if err != nil {
		// log
		return models.RepoState{}
	}

	err = bot.cli.SetRepoReviewer(
		org,
		repoName,
		sdk.SetRepoReviewer{
			Assignees:       " ", //TODO need set to " "?
			Testers:         " ",
			AssigneesNumber: 0,
			TestersNumber:   0,
		},
	)
	if err != nil {
		// log
	}

	branches := []community.RepoBranch{}
	for _, item := range repo.Branches {
		if item.Name == "master" {
			continue
		}

		if b, ok := bot.createBranch(org, repoName, item); ok {
			branches = append(branches, b)
		}
	}

	members := []string{}
	for _, item := range expectRepo.expectOwners {
		if err := bot.addRepoMember(org, repoName, item); err != nil {
			// log
		} else {
			members = append(members, item)
		}
	}

	return models.RepoState{
		Available: true,
		Branches:  branches,
		Members:   members,
		Property: models.RepoProperty{
			CanComment: repo.Commentable,
			Private:    repo.IsPrivate(),
		},
	}
}

func (bot *robot) renameRepo(org string, repo *community.Repository) models.RepoState {
	repoName := repo.Name

	err := bot.cli.UpdateRepo(
		org,
		repo.RenameFrom,
		sdk.RepoPatchParam{
			Name: repoName,
			Path: repoName,
		},
	)

	if err != nil {
		// log
		return models.RepoState{}
	}

	r := models.RepoState{Available: true}

	branches, err := bot.listAllBranchOfRepo(org, repoName)
	if err != nil {

	} else {
		r.Branches = branches
	}

	newRepo, err := bot.cli.GetGiteeRepo(org, repoName)
	if err != nil {

	} else {
		r.Members = newRepo.Members
		r.Property = models.RepoProperty{
			Private:    newRepo.Private,
			CanComment: newRepo.CanComment,
		}
	}

	return r
}

func (bot *robot) updateRepo(expectRepo *expectRepoInfo, lp models.RepoProperty) models.RepoProperty {
	org := expectRepo.org
	repo := expectRepo.expectRepoState

	ep := repo.IsPrivate()
	ec := repo.Commentable
	if ep != lp.Private || ec != lp.CanComment {
		err := bot.cli.UpdateRepo(
			org,
			repo.Name,
			sdk.RepoPatchParam{
				Name:       repo.Name,
				CanComment: strconv.FormatBool(ec),
				Private:    strconv.FormatBool(ep),
			},
		)
		if err == nil {
			return models.RepoProperty{
				Private:    ep,
				CanComment: ec,
			}
		}
		// log
	}
	return lp
}
