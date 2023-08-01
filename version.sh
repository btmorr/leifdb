#!/usr/bin/env bash

BRANCH=$(git rev-parse --abbrev-ref HEAD | sed 's/\//-/')
SHORT_HASH=$(git rev-parse --short HEAD)
VERSION="0.2.0-beta.2"

if [ $BRANCH == "main" ]
then
    echo $VERSION
else
    echo $VERSION+$BRANCH.$SHORT_HASH
fi
