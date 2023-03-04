# bazels3cache

This is an application that acts like a proxy between Bazel application and AWS S3 bucket.

# Installation

## Pre-build binaries are available for 5 major platforms:
https://github.com/krisukox/bazels3cache/releases/latest

- amd64 linux
- arm64 linux
- amd64 darwin
- arm64 darwin
- amd64 windows

#### Linux/Darwin

Choose your platform and install app under /usr/local/bin, e.g.:

`sudo wget https://github.com/krisukox/bazels3cache/releases/latest/bazels3cache-x86-linux -O /usr/local/bin/bazels3cache`


## Installation using Go:
If you have go installed the the app can be installed using go install command:
go install 
Then binary will be under `$GOPATH/bin`. Remember to add `$GOPATH/bin` to your `$PATH`.



The application is pre-build

### Benchmark

Benchamrk uses netem to simulate delay. 

Benchmark can be run with:

make run-benchmark
