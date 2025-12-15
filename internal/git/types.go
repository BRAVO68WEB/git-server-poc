package git

import "time"

type Repo struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type FileEntry struct {
	Mode string `json:"mode"`
	Type string `json:"type"`
	Hash string `json:"hash"`
	Name string `json:"name"`
}

type Commit struct {
	Hash    string    `json:"hash"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
}

type Branch struct {
	Name   string `json:"name"`
	IsHead bool   `json:"is_head"`
}

type Diff struct {
	Content string `json:"content"`
}

type BlameLine struct {
	LineNo  int       `json:"line_no"`
	Commit  string    `json:"commit"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Content string    `json:"content"`
}
