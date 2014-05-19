#!/bin/bash

if [ "$1" = "full" ] ; then
    go build -a github.com/gholt/blackfridaytext && \
    go install -a github.com/gholt/blackfridaytext/blackfridaytext-tool
else
    go build github.com/gholt/blackfridaytext && \
    go install github.com/gholt/blackfridaytext/blackfridaytext-tool
fi
