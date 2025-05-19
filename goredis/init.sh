#!/bin/bash
go build -o goredis
if [ $? -ne 0 ]; then
 echo "build fail"
 exit 1
fi
./goredis
