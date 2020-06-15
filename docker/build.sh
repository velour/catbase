#!/bin/bash -e

set -e
set -o pipefail

# To be run from the catbase src directory
[[ ! -f main.go ]] && echo "You must run this from the catbase src root." && exit 1

docker build -t velour/catbase -t chrissexton/private:catbase .
docker push chrissexton/private:catbase
