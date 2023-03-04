# bazels3cache

This application acts like a proxy between [Bazel](https://bazel.build/) build system and [AWS S3](https://aws.amazon.com/s3/).

## Installation (manual download):
The pre-build binaries are available for the following platforms:
- amd64 linux
- arm64 linux
- amd64 darwin
- arm64 darwin
- amd64 windows

#### Linux/Darwin:

Choose your platform from the [releases page](https://github.com/krisukox/bazels3cache/releases/) and install app under `/usr/local/bin`, e.g.:

`sudo wget https://github.com/krisukox/bazels3cache/releases/latest/download/bazels3cache-linux-amd64 -O /usr/local/bin/bazels3cache`

## Installation using Go:

`go install -v github.com/krisukox/bazels3cache@latest`  

## Testing

This project uses [s3ninja](https://s3ninja.net/) in order to simulate AWS S3 bucket.

### Integration test

Integration test:
- builds the test workspace
- cleans workspace
- builds it again
- check if artifacts was downloaded from the remote cache.

### Benchmark

Benchmark builds [Bazel](https://github.com/bazelbuild/bazel) project. It uses [netem](https://wiki.linuxfoundation.org/networking/netem) to simulate delay. Benchmark can be run with:  
`make run-benchmark`

Benchmark configuration environment variables:  
- `DELAY` - value in milliseconds that natem uses to simulate delay
- `BENCHMARK_TARGET` - available targets: `test_performance_1` and `test_performance_2`
- `BAZEL_TARGET` - target of the bazel repositorium that the benchmark will build

Default configuration is available [here](https://github.com/krisukox/bazels3cache/blob/main/test/benchmark.env).


