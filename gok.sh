#!/bin/sh
o=$(mktemp tmp.XXXXXXXXXX)

fail() {
	echo Failed
	cat $o | grep -v deprecated
	rm $o
	exit 1
}

trap fail INT TERM

echo gofmt
gofmt -l $(find . -name '*.go') > $o 2>&1
test $(wc -l $o | awk '{ print $1 }') = "0" || fail

echo govet
go vet ./... > $o 2>&1

echo go test
go test -test.timeout=60s ./... > $o 2>&1 || fail

rm $o
