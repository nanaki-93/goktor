package service

import (
	"fmt"

	"github.com/go-git/go-git"
)

type GitService interface {
	UpdateAllBranchesProject(path string) error
}
type GitModelService struct{}

func NewGitService() GitService {
	return &GitModelService{}
}

func (*GitModelService) UpdateAllBranchesProject(path string) error {

	repo, _ := git.PlainOpen(path)
	fmt.Println("")
	err := repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})
	return nil
}
