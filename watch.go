package main

import (
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
	"github.com/opensourceways/robot-gitee-repo-watcher/models"
)

func (bot *robot) run(watchingRepo, repoFilePath, sigFilePath, sigDir string) error {
	v := strings.Split(watchingRepo, "/")

	repos, err := bot.initExpectRepos(repoFileInfo)
	if err != nil {
		return err
	}

	actualRepos, err := bot.loadALLRepos(repos.repos.Community)
	if err != nil {
		return err
	}

	//return bot.watch(actualRepos,
}

func (bot *robot) stop() {

}

func (bot *robot) watch(actual *actualRepoState, repoFileInfo, sigFileInfo repoFile) error {
	repos, err := bot.initExpectRepos(repoFileInfo)
	if err != nil {
		return err
	}

	sigs, err := bot.initRepoSigs(sigFileInfo)
	if err != nil {
		return err
	}

	sigOwners := map[string]*expectSigOwners{}

	for {
		allFiles, err := bot.ListAllFilesOfRepo(
			repoFileInfo.org,
			repoFileInfo.repo,
			repoFileInfo.branch,
		)
		if err != nil {
			// log
		}

		if err := repos.update(allFiles[repoFileInfo.path]); err != nil {
			// log
		}

		repoMap := make(map[string]*community.Repository)
		v := repos.repos.Repositories
		for i := range v {
			item := &v[i]
			repoMap[item.Name] = item
		}

		if err := sigs.update(allFiles[sigFileInfo.path]); err != nil {
			// log
		}

		for _, sig := range sigs.sigs.Sigs {
			sigName := sig.Name
			sigOwnerFilePath := filepath.Join(
				filepath.Dir(sigFileInfo.path),
				sigName, "OWNERS",
			)

			sigOwner, ok := sigOwners[sigName]
			if !ok {
				sigOwner = &expectSigOwners{
					cli:  bot.cli,
					file: repoFileInfo,
				}
				sigOwner.file.path = sigOwnerFilePath
				sigOwners[sigName] = sigOwner
			}

			if err := sigOwner.update(allFiles[sigOwnerFilePath]); err != nil {
				//log
				// it should think about the unormal input data
			}

			for _, repoName := range sig.Repositories {
				task := &taskInfo{
					expectSigOwners: sigOwner.owners.Maintainers,
					expectRepoInfo:  repoMap[repoName],
					currentState:    actual.getOrNewARepo(repoName),
				}
				bot.execTask(task)

				delete(repoMap, repoName)
			}
		}

		for repoName, repo := range repoMap {
			task := &taskInfo{
				expectRepoInfo: repo,
				currentState:   actual.getOrNewARepo(repoName),
			}
			bot.execTask(task)
		}
	}
}

func (bot *robot) execTask(task *taskInfo) {
	f := func(isNewRepo bool, branches sets.String, members sets.String) (models.AfterUpdate, error) {
		if isNewRepo {
			return bot.handleRepo(task)
		}

		newBranches, err := bot.handleBranch(task)
		if err != nil {
			//log
			newBranches = branches.UnsortedList()
		}

		newMembers, err := bot.handleMember(task)
		if err != nil {
			//log
			newMembers = members.UnsortedList()
		}

		return models.AfterUpdate{
			Available: isNewRepo,
			Branches:  newBranches,
			Members:   newMembers,
		}, nil

	}

	err := pool.Submit(func() {
		task.currentState.Update(f)
	})
	if err != nil {
		//log
	}
}

func (bot *robot) handleRepo(task *taskInfo) (models.AfterUpdate, error) {

}

func (bot *robot) handleBranch(task *taskInfo) ([]string, error) {

}

func (bot *robot) handleMember(task *taskInfo) ([]string, error) {

}

func (bot *robot) ListAllFilesOfRepo(org, repo, branch string) (map[string]string, error) {
	trees, err := bot.cli.GetDirectoryTree(org, repo, branch, 1)
	if err != nil || len(trees.Tree) == 0 {
		return nil, err
	}

	r := make(map[string]string)
	for i := range trees.Tree {
		item := &trees.Tree[i]
		r[item.Path] = item.Sha
	}

	return r, nil
}
