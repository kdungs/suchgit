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
		"_templates/blob.html",
		"_templates/tree.html",
		"_templates/repos.html",
		"_templates/repo.html")
	if err != nil {
		panic(err)
	}

	sg.Router = mux.NewRouter()
	sg.Router.HandleFunc("/", sg.HandleIndex)
	sg.Router.HandleFunc("/{repo}/blob/{ref}/{file}", sg.HandleBlob)
	sg.Router.HandleFunc("/{repo}/blob/{file}", sg.HandleBlob) // defaults to ref=head
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
	// --- TESTING ---
	sg.Router.HandleFunc("/test/{repo}/file/{commit}/{filename}", sg.TestFileHandler)
	sg.Router.HandleFunc("/test/{repo}/diff/{id1}/{id2}", sg.TestDiffHandler)
	sg.Router.HandleFunc("/test/{repo}/{ref}", sg.TestHandler)
	sg.Router.HandleFunc("/test/{repo}", sg.TestHandler)
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

func (sg *SuchGit) HandleBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	repo, err := git.OpenRepository(filepath.Join(sg.RepoRoot, repoName))
	if err != nil {
		sg.ShowError(w, err)
		return
	}
	var oid *git.Oid
	refName, set := vars["ref"]
	if set {
		oid, err = git.NewOid(refName)
		if err != nil { // It's a reference!
			ref, err := repo.DwimReference(refName)
			if err != nil {
				sg.ShowError(w, err)
				return
			}
			oid = ref.Target()
		}
	} else {
		ref, err := repo.Head()
		if err != nil {
			sg.ShowError(w, err)
			return
		}
		oid = ref.Target()
	}
	commit, err := repo.LookupCommit(oid)
	if err != nil {
		sg.ShowError(w, err)
		return
	}
	tree, err := commit.Tree()
	if err != nil {
		sg.ShowError(w, err)
		return
	}
	fileEntry, err := tree.EntryByPath(vars["file"])
	if err != nil {
		sg.ShowError(w, err)
		return
	}
	if fileEntry.Type != git.ObjectBlob {
		fmt.Fprintf(w, "File has to be blob!\n")
		return
	}
	oid = fileEntry.Id
	file, err := repo.LookupBlob(oid)
	if err != nil {
		fmt.Fprintf(w, "326 Error: %s\n", err)
		return
	}
	sg.Tpl.ExecuteTemplate(w, "blob.html", string(file.Contents()))
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

func (sg *SuchGit) HandleCommits(w http.ResponseWriter, r *http.Request) {}

func (sg *SuchGit) HandleCommit(w http.ResponseWriter, r *http.Request) {}

func (sg *SuchGit) TestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	repo, err := git.OpenRepository(filepath.Join(sg.RepoRoot, repoName+".git"))
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}

	refName, refSet := vars["ref"]
	if refSet {
		ref, err := repo.DwimReference(refName)
		if err != nil {
			fmt.Fprintf(w, "Error: %s\n", err)
			return
		}
		fmt.Fprintf(w, "%s (%s)\n", ref.Shorthand(), ref.Name())
		oid := ref.Target()
		commit, err := repo.LookupCommit(oid)
		if err != nil {
			fmt.Fprintf(w, "Error: %s\n", err)
			return
		}
		for ; commit.ParentCount() > 0; commit = commit.Parent(0) {
			fmt.Fprintf(w, "[%s]\n%s\n\n", commit.Id().String(), commit.Message())
		}
	} else {
		refit, err := repo.NewReferenceIterator()
		if err != nil {
			fmt.Fprintf(w, "Error: %s\n", err)
			return
		}
		head, err := repo.Head()
		if err != nil {
			fmt.Fprintf(w, "Error: %s\n", err)
			return
		}
		for {
			ref, err := refit.Next()
			if err != nil {
				break // end of iterator
			}
			if ref.Name() == head.Name() {
				fmt.Fprintf(w, "*")
			} else {
				fmt.Fprintf(w, " ")
			}
			if ref.IsBranch() {
				fmt.Fprintf(w, "[Branch]")
			} else if ref.IsTag() {
				fmt.Fprintf(w, "[Tag]   ")
			} else if ref.IsRemote() {
				fmt.Fprintf(w, "[Remote]")
			} else {
				fmt.Fprintf(w, "--------")
			}
			fmt.Fprintf(w, " %25s (%s)\n", ref.Shorthand(), ref.Name())
		}

	}
}

func (sg *SuchGit) TestDiffHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	repo, err := git.OpenRepository(filepath.Join(sg.RepoRoot, repoName+".git"))
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}

	id1, err := git.NewOid(vars["id1"])
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	commit1, err := repo.LookupCommit(id1)
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	tree1, err := commit1.Tree()
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	//id2, err := git.NewOid(vars["id2"])
	//if err != nil {
	//	fmt.Fprintf(w, "Error: %s\n", err)
	//	return
	//}
	//commit2, err := repo.LookupCommit(id2)
	//if err != nil {
	//	fmt.Fprintf(w, "Error: %s\n", err)
	//	return
	//}
	//tree2, err := commit2.Tree()
	//if err != nil {
	//	fmt.Fprintf(w, "Error: %s\n", err)
	//	return
	//}
	defaultDiffOptions, err := git.DefaultDiffOptions()
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	diff, err := repo.DiffTreeToTree(nil, tree1, &defaultDiffOptions)
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	deltas, err := diff.NumDeltas()
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	for i := 0; i < deltas; i++ {
		patch, err := diff.Patch(i)
		if err != nil {
			fmt.Fprintf(w, "Error: %s\n", err)
			return
		}
		patchString, err := patch.String()
		if err != nil {
			fmt.Fprintf(w, "Error: %s\n", err)
			return
		}
		fmt.Fprintf(w, "%s\n", patchString)
	}
}

func (sg *SuchGit) TestFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	repo, err := git.OpenRepository(filepath.Join(sg.RepoRoot, repoName+".git"))
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}

	id1, err := git.NewOid(vars["commit"])
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	commit1, err := repo.LookupCommit(id1)
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	tree1, err := commit1.Tree()
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	fileEntry, err := tree1.EntryByPath(vars["filename"])
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
		return
	}
	if fileEntry.Type != git.ObjectBlob {
		fmt.Fprintf(w, "File has to be blob!\n")
		return
	}
	oid := fileEntry.Id
	file, err := repo.LookupBlob(oid)
	if err != nil {
		fmt.Fprintf(w, "326 Error: %s\n", err)
		return
	}
	fmt.Fprintf(w, "<pre>%s</pre>\n", file.Contents())
}
