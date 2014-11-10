package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/libgit2/git2go"
)

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func listRepos(path string) []string {
	repos, err := filepath.Glob(filepath.Join(path, "*.git"))
	panicOnError(err)
	for i, repo := range repos {
		repos[i] = filepath.Base(repo)
	}
	return repos
}

func listBranches(repo *git.Repository) []string {
	refit, err := repo.NewReferenceIterator()
	panicOnError(err)
	branches := make([]string, 0)
	for {
		ref, err := refit.Next()
		if err != nil {
			break // Reached end of iterator
		}
		if ref.IsBranch() {
			branches = append(branches, ref.Shorthand())
		}
	}
	return branches
}

func listFiles(folder string, tree *git.Tree) []string {
	files := make([]string, 0, tree.EntryCount())
	for i := uint64(0); i < tree.EntryCount(); i++ {
		files = append(files, tree.EntryByIndex(i).Name)
	}
	return files
}

type SuchGit struct {
	RepoRoot string
	Tpl      *template.Template
	Router   *mux.Router
}

func (g *SuchGit) Setup() {
	var err error
	g.Tpl, err = template.ParseFiles(
		"_templates/header.html",
		"_templates/footer.html",
		"_templates/error.html",
		"_templates/repos.html",
		"_templates/repo.html")
	panicOnError(err)

	g.Router = mux.NewRouter()

	g.Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := g.Tpl.ExecuteTemplate(w, "repos.html", listRepos(g.RepoRoot))
		panicOnError(err)
	})

	g.Router.HandleFunc("/{repo}", g.RepoHandler)
	g.Router.HandleFunc("/{repo}/{branch}", g.RepoHandler)
}

type RepoResponse struct {
	Name     string
	Branches []string
}

func (g *SuchGit) RepoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repo, err := git.OpenRepository(filepath.Join(g.RepoRoot, vars["repo"]))
	if err != nil {
		err := g.Tpl.ExecuteTemplate(w, "error.html", err)
		panicOnError(err)
	} else {
		resp := RepoResponse{vars["repo"], listBranches(repo)}
		err = g.Tpl.ExecuteTemplate(w, "repo.html", resp)
		panicOnError(err)
	}
}

func main() {
	gitorade := SuchGit{"_repos", nil, nil}
	gitorade.Setup()
	http.Handle("/", gitorade.Router)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
