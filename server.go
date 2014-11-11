package main

import (
	"log"
	"net/http"
)

func main() {
	sg := NewSuchGit("_repos")
	http.Handle("/", sg.Router)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
