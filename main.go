// Copyright (c) 2013, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/goulash/pr"
	"github.com/ogier/pflag"
)

const outputTmpl = `GitHub @y{{.User}}/{{.Name}}@|
{{range .Releases}}@!{{.Tag}}:@| ({{.Name}}){{range .Assets}}
	@r{{.Name}}@|: {{.DownloadCount}}
{{end}}{{end}}
`

type GithubRepo struct {
	User     string
	Name     string
	Releases []struct {
		Tag    string `json:"tag_name"`
		Name   string `json:"name"`
		Assets []struct {
			Name          string `json:"name"`
			DownloadCount int    `json:"download_count`
		} `json:"assets"`
	}
}

// FindGithubInfo tries to find the username and repository
// of the current git project.
//
// We do this using git remote -v. If it works, we should get something like:
//
//	origin	git@github.com:cassava/repoctl.git (fetch)
//	origin	git@github.com:cassava/repoctl.git (push)
//
// This is one of multiple formats. We need to expand this with time to cover
// all the different formats that it can have.
//
//  1. Check if github is there
//  2. Check protocol and extract information
//		a) git protocol: `git@github.com:username/repository.git`
//		b) https protocol: `https://github.com/username/repository.git`
//
func FindGithubInfo(_ string) (*GithubRepo, error) {
	// TODO: implement this
	return &GithubRepo{
		User: "cassava",
		Name: "repoctl",
	}, nil
}

// GetReleaseInfo gets the release information
func GetReleaseInfo(gi *GithubRepo) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", gi.User, gi.Name)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&gi.Releases)
	if err != nil {
		return err
	}
	return nil
}

func dieIfFatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func main() {
	color := pflag.String("color", "auto", "whether to use color (always|auto|never)")
	pflag.Parse()

	colorizer := pr.NewColorizer()
	if *color == "auto" {
		colorizer.SetFile(os.Stdout)
	} else if *color == "always" {
		colorizer.SetEnabled(true)
	} else if *color == "never" {
		colorizer.SetEnabled(false)
	}

	wd, err := os.Getwd()
	dieIfFatal(err)
	gr, err := FindGithubInfo(wd)
	dieIfFatal(err)
	err = GetReleaseInfo(gr)
	dieIfFatal(err)

	t := template.Must(template.New("output").Parse(colorizer.Sprint(outputTmpl)))
	t.Execute(os.Stdout, gr)
}
