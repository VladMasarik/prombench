package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "benchmark result posting and formating tool.\n-i location of github hook file (even.json)")
	app.HelpFlag.Short('h')
	input := app.Flag("input", "path to event.json").Short('i').Default("/github/workflow/event.json").String()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	data, err := ioutil.ReadFile(*input)
	if err != nil {
		log.Fatalln(err)
	}

	// Temporary fix for the new Github actions time format. This makes the time stamps unusable.
	txt := string(data)
	reg := regexp.MustCompile("(.*)\"[0-9]+/[0-9]+/2019 [0-9]+:[0-9]+:[0-9]+ [AP]M(.*)")
	txt = reg.ReplaceAllString(txt, "$1\"2019-06-11T09:26:28Z$2")
	data = []byte(txt)
	// End of the temporary fix

	event, err := github.ParseWebHook("issue_comment", data)
	if err != nil {
		log.Fatalln(err)
	}

	switch e := event.(type) {
	case *github.IssueCommentEvent:

		//Variable and environment setup.
		branch := readFile("/github/home/ARG_0")
		issueComment := readFile("/github/home/ARG_1")
		race := readFile("/github/home/ARG_2")
		prNumber, err := strconv.Atoi(readFile("/github/home/ARG_3"))
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("ARG_0 / branch = %s", branch)
		log.Printf("ARG_1 / issueComment = %s", issueComment)
		log.Printf("ARG_2 / race = %s", race)
		log.Printf("ARG_3 / prnumber = %d", prNumber)

		if err := os.Setenv("GO111MODULE", "on"); err != nil {
			log.Fatalln(err)
		}

		clt := newClient(os.Getenv("GITHUB_TOKEN"))
		home := os.Getenv("HOME")
		oldPath := strings.Join([]string{home, "/old.txt"}, "")
		newPath := strings.Join([]string{home, "/new.txt"}, "")
		owner := *e.GetRepo().Owner.Login
		repo := *e.GetRepo().Name

		log.Printf("Owner %s", owner)
		log.Printf("Repo %s", repo)

		os.Chdir(os.Getenv("GITHUB_WORKSPACE"))
		data, err := exec.Command("git", "clone", fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)).CombinedOutput()
		log.Println(string(data))
		if err != nil {
			log.Fatalln(err)
		}
		os.Chdir(repo)

		data, err = exec.Command("git", "log", "--pretty=format:\"%H\"", "-n 1").CombinedOutput()
		latestCommitHash := string(data)
		log.Println(latestCommitHash)
		if err != nil {
			log.Fatalln(err)
		}

		data, err = exec.Command("git", "config", "--global", "user.email", "prombench@example.com").CombinedOutput()
		if err != nil {
			log.Println(string(data))
			log.Fatalln(err)
		}
		data, err = exec.Command("git", "config", "--global", "user.name", "Prombench Bot Junior").CombinedOutput()
		if err != nil {
			log.Println(string(data))
			log.Fatalln(err)
		}

		// Branch setup.
		data, err = exec.Command("git", "fetch", "origin", fmt.Sprintf("pull/%d/head:pullrequest", prNumber)).CombinedOutput()
		if err != nil {
			log.Println(string(data))
			log.Fatalln(err)
		}

		data, err = exec.Command("git", "checkout", branch).CombinedOutput()
		if err != nil {
			log.Println(string(data))
			log.Fatalln(err)
		}

		cmd := exec.Command("git", "merge", "--squash", "--no-commit", "pullrequest")
		data, err = cmd.CombinedOutput()
		if err != nil {
			log.Println(string(data))
			log.Fatalln(err)
		}
		if cmd.ProcessState.ExitCode() != 0 {
			if err := postComment(clt, owner, repo, prNumber, "Git merge failed"); err != nil {
				log.Fatalln(err)
			}
		}

		data, err = exec.Command("git", "reset").CombinedOutput()
		if err != nil {
			log.Println(string(data))
			log.Fatalln(err)
		}

		// Benchmark the with pullrequest changes.
		if race == "-no-race" {
			cmd = exec.Command("go", "test", "-bench", issueComment, "-benchmem", "-v", "./...")
		} else {
			cmd = exec.Command("go", "test", "-bench", issueComment, "-benchmem", "-race", "-v", "./...")
		}
		data, err = cmd.CombinedOutput()
		log.Println(string(data))
		if err != nil || cmd.ProcessState.ExitCode() != 0 {
			if err := postComment(clt, owner, repo, prNumber, fmt.Sprintf("Go test for this pull request failed. Check https://github.com/%s/%s/commit/%s/checks for more information.", owner, repo, latestCommitHash)); err != nil {
				log.Fatalln(err)
			}
			log.Fatalln(err)
		}
		err = ioutil.WriteFile(newPath, data, 0644)
		if err != nil {
			log.Fatalln(err)
		}

		// Checkout to the comparing branch.
		filepath.Walk(os.Getenv("GITHUB_WORKSPACE"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || strings.Contains(path, ".git/") || strings.HasSuffix(info.Name(), "test.go") {
				return nil
			}

			data, err = exec.Command("git", "checkout", "--", path).CombinedOutput()
			if err != nil {
				log.Println(string(data))
				log.Fatalln(err)
			}
			return nil
		})

		// Benchmark the comparing branch.
		if race == "-no-race" {
			cmd = exec.Command("go", "test", "-bench", issueComment, "-benchmem", "-v", "./...")
		} else {
			cmd = exec.Command("go", "test", "-bench", issueComment, "-benchmem", "-race", "-v", "./...")
		}
		data, err = cmd.CombinedOutput()
		log.Println(string(data))
		if err != nil || cmd.ProcessState.ExitCode() != 0 {
			if err := postComment(clt, owner, repo, prNumber, fmt.Sprintf("Go test on branch %s failed. Check https://github.com/%s/%s/commit/%s/checks for more information.", branch, owner, repo, latestCommitHash)); err != nil {
				log.Fatalln(err)
			}
			log.Fatalln(err)
		}
		err = ioutil.WriteFile(oldPath, data, 0644)
		if err != nil {
			log.Fatalln(err)
		}

		data, err = exec.Command(strings.Join([]string{os.Getenv("GOPATH"), "/bin/benchcmp"}, ""), oldPath, newPath).CombinedOutput()
		if err != nil {
			log.Println(string(data))
			if err := postComment(clt, owner, repo, prNumber, fmt.Sprintf("Error: `benchcmp` failed. Check https://github.com/%s/%s/commit/%s/checks for more information.", owner, repo, latestCommitHash)); err != nil {
				log.Fatalln(err)
			}
			log.Fatalln(err)
		}
		if strings.Count(string(data), "\n") < 2 {
			if err := postComment(clt, owner, repo, prNumber, fmt.Sprintf("Error: `go test` could not match any benchmarked functions. Check https://github.com/%s/%s/commit/%s/checks for more information.", owner, repo, latestCommitHash)); err != nil {
				log.Fatalln(err)
			}
		}

		tableContent := strings.Split(string(data), "\n")
		// for i, e := range tableContent {
		for i := 0; i <= len(tableContent)-1; i++ {
			e := tableContent[i]
			log.Println(i)
			log.Printf("Before=%s\n", e)
			log.Println(tableContent)
			log.Println(tableContent[:i])
			log.Println(tableContent[i:])
			switch {
			case e == "":
				// Delete i-th element
				// tableContent = append(tableContent[:i], tableContent[i+1:]...)
				// continue outerForLoop

			case strings.Contains(e, "old ns/op"):
				e = "| Benchmark | Old ns/op | New ns/op | Delta |"
				tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

			case strings.Contains(e, "old MB/s"):
				e = "| Benchmark | Old MB/s | New MB/s | Speedup |"
				tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

			case strings.Contains(e, "old allocs"):
				e = "| Benchmark | Old allocs | New allocs | Delta |"
				tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

			case strings.Contains(e, "old bytes"):
				e = "| Benchmark | Old bytes | New bytes | Delta |"
				tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

			default:
				// Replace spaces with |
				e = strings.Join(strings.Fields(e), "|")
			}
			log.Println(tableContent)
			log.Println(tableContent[:i])
			log.Println(tableContent[i:])
			tableContent[i] = e
			log.Printf("After=%s\n", e)
		}
		// Insert into slice
		// tableContent = append(tableContent[:1], append([]string{"|-|-|-|-|"}, tableContent[1:]...)...)

		// Post the resulting comment.
		if err := postComment(clt, owner, repo, prNumber, strings.Join(tableContent, "\n")); err != nil {
			log.Fatalln(err)
		}

	default:
		log.Fatalln("simpleargs only supports issue_comment event")
	}
}

func newClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	clt := github.NewClient(tc)
	return clt
}

func readFile(name string) string {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		log.Fatalln(err)
	}
	return string(data)

}

func postComment(client *github.Client, owner string, repo string, prNumber int, comment string) error {
	issueComment := &github.IssueComment{Body: github.String(comment)}
	_, _, err := client.Issues.CreateComment(context.Background(), owner, repo, prNumber, issueComment)
	return err
}
