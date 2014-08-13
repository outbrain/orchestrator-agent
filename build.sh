#!/bin/bash

# Simple packaging of orchestrator-agent
#
# Requires fpm: https://github.com/jordansissel/fpm
#

release_version="1"
release_dir=/tmp/orchestrator-agent-release
release_files_dir=$release_dir/orchestrator-agent
rm -rf $release_dir/*
mkdir -p $release_files_dir/

cd  $(dirname $0)
rsync -av ./conf $release_files_dir/
GOPATH=/usr/share/golang:$(pwd)
go build -o $release_files_dir/orchestrator-agent ./src/github.com/outbrain/orchestrator-agent/main.go

if [[ $? -ne 0 ]] ; then
	exit 1
fi

cd $release_dir
# tar packaging
tar cfz orchestrator-agent-"${release_version}".tar.gz ./orchestrator-agent
# rpm packaging
fpm -v "${release_version}" -f -s dir -t rpm -n orchestrator-agent -C $release_dir --prefix=/usr/local ./orchestrator-agent
fpm -v "${release_version}" -f -s dir -t deb -n orchestrator-agent -C $release_dir --prefix=/usr/local ./orchestrator-agent

echo "---"
echo "Done. Find releases in $release_dir"
