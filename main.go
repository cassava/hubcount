// Copyright (c) 2016, Ben Morgan. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/goulash/color"
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
			DownloadCount int    `json:"download_count"`
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
	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return nil, err
	}

	r := regexp.MustCompilePOSIX(`([^\t]+)\t(git@|http://|https://)github\.com[/:]([^/]+)/(.+)\.git[ \t]*.*`)
	infos := make(map[string][2]string)
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		s := scanner.Text()
		if !strings.Contains(s, "github") {
			continue
		}
		if !r.MatchString(s) {
			fmt.Fprintf(os.Stderr, "Warning: cannot match %q.\n", s)
			continue
		}

		ls := r.FindStringSubmatch(s)
		source, _, user, repo := ls[1], ls[2], ls[3], ls[4]
		infos[source] = [2]string{user, repo}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var up [2]string
	if len(infos) == 0 {
		return nil, errors.New("not a GitHub repository")
	} else if len(infos) == 1 {
		for _, v := range infos {
			up = v
		}
	} else {
		// Try the origin repository first; if it exists, it is bound to be the right one.
		if v, ok := infos["origin"]; ok {
			up = v
		} else {
			for k, v := range infos {
				fmt.Fprintf(os.Stderr, "Warning: multiple GitHub repositories; using source %q.\n", k)
				up = v
				break
			}
		}
	}

	return &GithubRepo{
		User: up[0],
		Name: up[1],
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

func mustNot(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func main() {
	color := color.New()
	pflag.Var(color, "color", "when to use color (always|auto|never)")
	pflag.Parse()

	wd, err := os.Getwd()
	mustNot(err)
	gr, err := FindGithubInfo(wd)
	mustNot(err)
	err = GetReleaseInfo(gr)
	mustNot(err)

	t := template.Must(template.New("output").Parse(color.Sprint(outputTmpl)))
	t.Execute(os.Stdout, gr)
}
