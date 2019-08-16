## Intro
This tools is meant to be used as a Github action. The action itself is, to a large degree, unusable alone, as you need to combine it with another Github action that will provide necessary files to it. At this time, the only action it is supposed to work with, is [comment-monitor](https://github.com/prometheus/prombench/tree/master/tools/commentMonitor).

## Use
After a successful set up, you can create a comment in a pull request to execute the benchmarks. The form of the comment is following:
`/benchmark [branch] <golang test regex>  [-no-race]`
If no branch is specified, `master` branch is used. The golang test regex is used to execute benchmarks that match the regex. By defualt the benchmarks run with `-race` flag enabled, disable this by specifying `-no-race` in the comment. After a successful run, a table with results will be displayed in the comment section.

## Set up
- Create Github actions workflow file that is executed when an issue comment is created, `on = "issue_comment"`.
- Add comment-monitor Github action as a first step.
- Specify this regex `^/benchmark ?([^ B\.]+)? ?(\.|Bench.*|[^ ]+)? ?(-no-race)?.*$` in the `args` field of the comment-monitor.
- Specify this Github action as a pre-built image, build from this source code, or just refer to this repository from the workflow file.
- Provide a Github token as an environment variable to both comment-monitor and funcbench.

## How to build
`docker build -t <tag of your choice> .`

## Examples
### Example comments
Execute all benchmarks matching `FuncName` regex, and compare it with `master` branch.
 - `/benchmark master FuncName.*`

Execute all benchmarks, and compere the results with `devel` branch.
 - `/benchmark devel .`

### Example Github actions workflow file
```
on: issue_comment
name: Benchmark
jobs:
  commentMonitor:
    runs-on: ubuntu-latest
    steps:
    - name: commentMonitor
      uses: docker://prombench/comment-monitor:latest
      env:
        COMMENT_TEMPLATE: 'The benchmark has started.'
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      with:
        args: '"^/benchmark ?([^ ]+)? ?([^ ]+)? ?(-no-race)?.*$"'
    - name: benchmark
      uses: docker://prombench/funcbench:latest
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## What it does [SHORT]
It acquires arguments for `go test` benchmarks, executes and compares the benchmarks based on the provided arguments. Then it posts results into pull request it was called from.

## What it does [LONG]
Comparison is done using [benchcmp](https://godoc.org/golang.org/x/tools/cmd/benchcmp). Arguments are read from files created by previous action (for example commentMonitor), which is responsible for the argument parsing.