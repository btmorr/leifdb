#!/usr/bin/env bash

BRANCH=$(git rev-parse --abbrev-ref HEAD)
SHORT_HASH=$(git rev-parse --short HEAD)
VERSION="0.1.0-beta"

if [ $BRANCH == "edge" ]
then
    echo $VERSION
else
    echo $VERSION+$BRANCH.$SHORT_HASH
fi
