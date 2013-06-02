#!/bin/bash
VER=$1
mkdir ./build/
mkdir -p ./build/$VER/win/etc
mkdir -p ./build/$VER/mac/etc

cp -r ../data/ ./build/$VER/win/
cp -r ./etc/ ./build/$VER/win/

cp -r ../data/ ./build/$VER/mac/
cp -r ./etc/ ./build/$VER/mac/

go build -o ./build/$VER/win/AlDftr.exe AlDftr.go

# Mac Version
export CGO_ENABLED=0
export GOOS=darwin
go build -o ./build/$VER/mac/AlDftr AlDftr.go

# Return windows env vars
export CGO_ENABLED=1
export GOOS=windows

# Create zip files

cd ./build/$VER/win/
zip -r ../AlDftr-win-$VER.zip *
cd -

cd ./build/$VER/mac/
zip -r ../AlDftr-mac-$VER.zip *
cd -