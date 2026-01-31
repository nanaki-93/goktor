package service

import (
	"errors"
	"fmt"
	"path"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type GitService interface {
	UpdateAllBranchesProject(path string) error
	UpdateRemote(path string, newRemote string) error
}
type GitModelService struct{}

func NewGitService() GitService {
	return &GitModelService{}
}

func (*GitModelService) UpdateAllBranchesProject(path string) error {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("failed to open repo: %v", err)
	}

	// Fetch from origin
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Progress:   nil,
		Force:      true,
		Tags:       git.AllTags,
		// Add Auth if needed
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch failed: %v", err)
	}

	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("failed to list branches: %v", err)
	}

	err = branches.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		fmt.Printf("Found branch: %s\n", branchName)
		// You can add logic to update/reset branches here
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (*GitModelService) UpdateRemote(folderPath string, newRemote string) error {
	repo, err := git.PlainOpen(folderPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %v", err)
	}
	fmt.Println("updating remote for the repository:", folderPath)

	remotes, err := repo.Remotes()
	if err != nil {
		return fmt.Errorf("failed to list remotes: %v", err)
	}
	var found bool
	for _, r := range remotes {
		if r.Config().Name == "origin" {
			found = true

			oldRemote, err := updateRemote(newRemote, repo)
			if err != nil {
				return fmt.Errorf("failed to change remote: %v", err)
			}

			cfg, err := repo.Storer.Config()
			if err != nil {
				return fmt.Errorf("failed to get config: %v", err)
			}
			remoteCfg, _ := cfg.Remotes["origin"]
			fmt.Println("remote url from updated config:" + remoteCfg.URLs[0])

			err = repo.Fetch(&git.FetchOptions{
				RemoteName: "origin",
				Force:      true,
			})
			if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				fmt.Println("fetch failed, rolling back to old remote", err)
				_, rollErr := updateRemote(oldRemote, repo)
				if rollErr != nil {
					fmt.Println("rollback failed, check the remote manually:", rollErr)
				}
				return fmt.Errorf("fetch failed, rollback completed: %v", err)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("remote 'origin' not found")
	}

	return nil
}

func updateRemote(newRemote string, repo *git.Repository) (string, error) {
	cfg, err := repo.Storer.Config()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %v", err)
	}
	remoteCfg, ok := cfg.Remotes["origin"]
	oldRemote := remoteCfg.URLs[0]

	fmt.Println("old remote value:", oldRemote)
	projectName := path.Base(oldRemote)
	oldRemoteBase := oldRemote[:len(oldRemote)-len(projectName)]
	if len(oldRemoteBase) > 0 && oldRemoteBase[len(oldRemoteBase)-1] == '/' {
		oldRemoteBase = oldRemoteBase[:len(oldRemoteBase)-1]
	}
	projectName = projectName[:len(projectName)-4] // remove .git suffix
	fmt.Println("project name:", projectName)
	if !ok {
		return "", fmt.Errorf("remote 'origin' not found in config")
	}
	remoteCfg.URLs = []string{newRemote + "/" + projectName + ".git"}
	fmt.Println("new remote value:", remoteCfg.URLs)
	err = repo.Storer.SetConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to set config: %v", err)
	}
	return oldRemoteBase, nil
}
