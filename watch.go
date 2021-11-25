package main

import (
	"context"
	"time"

	"github.com/opensourceways/robot-gitee-repo-watcher/community"
	"github.com/opensourceways/robot-gitee-repo-watcher/models"
)

type expectRepoInfo struct {
	expectRepoState *community.Repository
	expectOwners    []string
	org             string
}

func (bot *robot) run(ctx context.Context, opt *options) error {
	w, _ := opt.parseWatchingRepo()

	expect := &expectState{
		w:   w,
		cli: bot.cli,
	}

	org, err := expect.init(opt.repoFilePath, opt.sigFilePath, opt.sigDir)
	if err != nil {
		return err
	}

	local, err := bot.loadALLRepos(org)
	if err != nil {
		return err
	}

	bot.watch(ctx, org, local, expect, opt.interval)
	return nil
}

func (bot *robot) watch(ctx context.Context, org string, local *localState, expect *expectState, interval int) {
	if interval <= 0 {
		for {
			if isCancelled(ctx) {
				break
			}

			bot.checkOnce(ctx, org, local, expect)
		}
	} else {
		t := time.Duration(interval) * time.Minute

		for {
			if isCancelled(ctx) {
				break
			}

			s := time.Now()

			bot.checkOnce(ctx, org, local, expect)

			e := time.Now()
			if v := e.Sub(s); v < t {
				time.Sleep(t - v)
			}
		}
	}

	bot.wg.Wait()
}

func (bot *robot) checkOnce(ctx context.Context, org string, local *localState, expect *expectState) {
	f := func(repo *community.Repository, owners []string) {
		if repo == nil {
			return
		}

		bot.execTask(
			local.getOrNewRepo(repo.Name),
			expectRepoInfo{
				org:             org,
				expectOwners:    owners,
				expectRepoState: repo,
			},
		)
	}

	isStopped := func() bool {
		return isCancelled(ctx)
	}

	expect.check(isStopped, f)
}

func (bot *robot) execTask(localRepo *models.Repo, expectRepo expectRepoInfo) {
	f := func(before models.RepoState) models.RepoState {
		if !before.Available {
			return bot.createRepo(expectRepo)
		}

		return models.RepoState{
			Available: true,
			Branches:  bot.handleBranch(expectRepo, before.Branches),
			Members:   bot.handleMember(expectRepo, before.Members),
			Property:  bot.updateRepo(expectRepo, before.Property),
		}
	}

	bot.wg.Add(1)
	err := bot.pool.Submit(func() {
		defer bot.wg.Done()
		localRepo.Update(f)
	})
	if err != nil {
		bot.wg.Done()
		//log
	}
}

func isCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
