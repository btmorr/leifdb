#!/usr/bin/env bash

BRANCH=$(git rev-parse --abbrev-ref HEAD | sed 's/\//-/')
SHORT_HASH=$(git rev-parse --short HEAD)
VERSION="0.2.0-beta.0"

if [ $BRANCH == "edge" ]
then
    echo $VERSION
else
    echo $VERSION+$BRANCH.$SHORT_HASH
fi
