#!/bin/bash
set e
docker build -t docker.io/prombench/funcbench .
docker push docker.io/prombench/funcbench

curl -X POST -H "Authorization: token $GHTOKEN" -d '{"body":"/benchmark oldActions BenchmarkStringBytesEquals/1-flip-end-by.*"}' https://api.github.com/repos/prometheus-community/ga2/issues/1/comments
