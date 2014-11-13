/**
What about https://stackoverflow.com/questions/20932078/where-are-the-project-files-stored-in-a-git-repository-git-folder
*/

package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/libgit2/git2go"
)

type SuchGit struct {
	RepoRoot string
	Tpl      *template.Template
	Router   *mux.Router
}

func NewSuchGit(repoRoot string) *SuchGit {
	sg := new(SuchGit)
	sg.RepoRoot = repoRoot
	sg.Setup()
	return sg
}

func (sg *SuchGit) Setup() {
	var err error
	sg.Tpl, err = template.ParseFiles(
		"_templates/header.html",
		"_templates/footer.html",
		"_templates/error.html",
		"_templates/tree.html",
		"_templates/repos.html",
		"_templates/repo.html")
	if err != nil {
		panic(err)
	}

	sg.Router = mux.NewRouter()
	sg.Router.HandleFunc("/", sg.HandleIndex)
	sg.Router.HandleFunc("/{repo}/tree/{ref}/{folder}", sg.HandleTree)
	sg.Router.HandleFunc("/{repo}/tree/{ref}", sg.HandleTree) // defaults to tree=/
	sg.Router.HandleFunc("/{repo}/tree/", sg.HandleTree)      // defaults to ref=head
	sg.Router.HandleFunc("/{repo}", sg.HandleTree)            // same as above
	sg.Router.HandleFunc("/{repo}/blob/{ref}/{file}", sg.HandleBlob)
	sg.Router.HandleFunc("/{repo}/blob/{file}", sg.HandleBlob) // defaults to ref=head
	sg.Router.HandleFunc("/{repo}/commits/{ref}", sg.HandleCommits)
	sg.Router.HandleFunc("/{repo}/commits", sg.HandleCommits) // defaults to ref=head
	sg.Router.HandleFunc("/{repo}/commit/{ref}", sg.HandleCommit)
	sg.Router.HandleFunc("/{repo}/commit", sg.HandleCommit) // defaults to ref=head
}

func (sg *SuchGit) ShowError(w http.ResponseWriter, err error) {
	err_ := sg.Tpl.ExecuteTemplate(w, "error.html", err)
	if err_ != nil {
		fmt.Fprintf(w, "Error: %s", err_)
		fmt.Fprintf(w, "From Error: %s", err)
	}
}

func (sg *SuchGit) ListRepositories() []string {
	repos, err := filepath.Glob(filepath.Join(sg.RepoRoot, "*.git"))
	if err != nil {
		return make([]string, 0)
	}
	for i, repo := range repos {
		repos[i] = filepath.Base(repo)
	}
	return repos
}

func (sg *SuchGit) HandleIndex(w http.ResponseWriter, r *http.Request) {
	repos := sg.ListRepositories()
	err := sg.Tpl.ExecuteTemplate(w, "repos.html", repos)
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
	}
}

func (sg *SuchGit) HandleTree(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	refName, set := vars["ref"]
	if !set {
		refName = "master"
	}
	folderName, set := vars["folder"]
	if !set {
		folderName = ""
	}

	repo, err := git.OpenRepository(filepath.Join(sg.RepoRoot, repoName))
	if err != nil {
		sg.ShowError(w, err)
		return
	}

	ref, err := repo.DwimReference(refName)
	if err != nil {
		sg.ShowError(w, err)
		return
	}

	oid := ref.Target()
	commit, err := repo.LookupCommit(oid)
	if err != nil {
		sg.ShowError(w, err)
		return
	}

	folder, err := commit.Tree()
	if err != nil {
		sg.ShowError(w, err)
		return
	}

	if folderName != "" {
		folderEntry, err := folder.EntryByPath(folderName)
		if err != nil {
			sg.ShowError(w, err)
			return
		}
		oid = folderEntry.Id
		folder, err = repo.LookupTree(oid)
		if err != nil {
			sg.ShowError(w, err)
			return
		}
	}

	count := folder.EntryCount()
	files := make([]string, 0, count)
	for i := uint64(0); i < count; i++ {
		files = append(files, folder.EntryByIndex(i).Name)
	}

	resp := TreeResponse{RepositoryResponse{repoName, refName, make([]string, 0), commit}, folderName, files}
	err = sg.Tpl.ExecuteTemplate(w, "tree.html", resp)
	if err != nil {
		sg.ShowError(w, err)
	}
}

func (sg *SuchGit) HandleBlob(w http.ResponseWriter, r *http.Request) {}

func (sg *SuchGit) HandleCommits(w http.ResponseWriter, r *http.Request) {}

func (sg *SuchGit) HandleCommit(w http.ResponseWriter, r *http.Request) {}
