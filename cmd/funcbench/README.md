## Intro
This repository contains source code for Github action. The action itself is, to a large degree, unusable alone, as you need to combine it with another action that will provide necessary files to it. At this time, the only action it is supposed to work with is commentMonitor, see the examples.

## What it does [QUICK]
It acquires arguments for `go test` benchmarks, and executes and compares the benchmarks based on the provided arguments. Then it posts results into the specified issue.

## Set up
Create Github actions workflow file and specify this Github action. Provide a Github token as an environment variable in the workflow file located inside `/.github` folder, and use the image build from this source code (or just refer to this repository from the workflow file). Additionally, specify that the Github action should be started when an issue comment is created. Github perceives pull requests as issues, therefore, you should use `on = "issue_comment"` event trigger.

## Use
If the set up was successful, you can comment `/benchmark <branch> <golang test regex> [-race]` in the pull request to execute the benchmarks. For example `/benchmark master . -race` will match all benchmarks, while testing for race condition, and the results of the pull request will be compared against the master branch.

## How to build
`docker build -t <tag of your choice> .`

## Examples
The complementary action examples:
- [commentMonitor](https://github.com/prometheus/prombench/tree/master/cmd/tools/commentMonitor)
- close copy of [commentMonitor](https://github.com/prometheus-community/yetAnotherTestGithubActions/tree/master/actions/main)

Example action workflows:
- [uses custom image that is build from commentMonitor](https://github.com/prometheus-community/yetAnotherTestGithubActions/blob/9fd48fced9dec8f5ecf92fce1e532c0d40508641/.github/main.workflow)
- [should work, but I never tested it. Also, build from commentMonitor](https://github.com/prometheus-community/github_actions/blob/bbba2f5ffc300914191730f9cfc9c1c1df694306/.github/main.workflow)

Example pull request use:
- https://github.com/prometheus-community/yetAnotherTestGithubActions/pull/8 ( Scroll down to seethe most recent results. )

## What it does [LONG]
Comparison is done using [benchcmp](https://godoc.org/golang.org/x/tools/cmd/benchcmp). Arguments are read from files created by previous action (for example commentMonitor), which is responsible for the argument parsing.