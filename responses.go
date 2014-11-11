package main

import (
	"github.com/libgit2/git2go"
)

type RepositoryResponse struct {
	Name   string
	Ref    string
	Refs   []string
	Commit *git.Commit
}

type TreeResponse struct {
	Repository RepositoryResponse
	Root       string
	Files      []string
}
