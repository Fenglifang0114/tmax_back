#!/bin/bash
#should change the version v1.1.0 when modify the source code
go build -v -ldflags="-X 'main.Version=v1.77' -X 'tmaxsrv/build.User=$(id -u -n)' -X 'tmaxsrv/build.Time=$(date)'"
