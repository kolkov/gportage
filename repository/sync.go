package repository

import (
    "github.com/go-git/go-git/v5"
)

type RepoManager struct {
    repos map[string]*git.Repository
}

func (rm *RepoManager) Sync(repoName string) error {
    repo, ok := rm.repos[repoName]
    if !ok {
        return errors.New("repository not found")
    }
    
    worktree, _ := repo.Worktree()
    return worktree.Pull(&git.PullOptions{
        Depth: 1,  // Поверхностное клонирование
    })
}