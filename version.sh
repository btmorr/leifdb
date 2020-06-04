#!/usr/bin/env bash

BRANCH=$(git rev-parse --abbrev-ref HEAD)
VERSION="0.1.0b"

if [ $BRANCH == "edge" ]
then
    echo $VERSION
else
    echo $VERSION-$BRANCH
fi
