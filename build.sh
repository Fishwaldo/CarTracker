#!/bin/sh
go build -ldflags="-X 'main.VersionSummary=`git-semver`'"