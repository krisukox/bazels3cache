# bazels3cache

This application acts like a proxy between the [Bazel](https://bazel.build/) build system and [AWS S3](https://aws.amazon.com/s3/). It supports the GET and PUT HTTP request methods. The application logs to the `~/.bazels3cache.log` file.
> **_NOTE:_** Currently only POSIX systems are supported.

## Start application

This command:
```
bazels3cache -bucket bucket_name
```

starts the server in the background. The default port is `7777`. If you want to use a different port, specify it with the `-port` switch.
```
bazels3cache -bucket bucket_name -port 5555
```

If the application is running, you can use it as a remote cache by passing the `--remote_cache=http://localhost:7777` flag to Bazel, e.g.:
```
bazel build //... --remote_cache=http://localhost:7777
```


## Stop application

```
bazels3cache -stop
```
or
```
curl http://localhost:7777/shutdown
```

## Installation (pre-built binaries)
The pre-built binaries are available for the following platforms:
- amd64 linux
- arm64 linux
- amd64 darwin
- arm64 darwin

#### Linux/Darwin

Choose your platform from the [releases page](https://github.com/krisukox/bazels3cache/releases/) and install the application under `/usr/local/bin`, e.g.:

```
sudo wget https://github.com/krisukox/bazels3cache/releases/latest/download/bazels3cache-linux-amd64 -O /usr/local/bin/bazels3cache
sudo chmod +x /usr/local/bin/bazels3cache
hash -r
```

## Installation using Go

```
go install -v github.com/krisukox/bazels3cache@HEAD
```

## Testing

This project uses [s3ninja](https://s3ninja.net/) in order to simulate AWS S3 bucket.

### Integration test

Integration test builds the [test workspace](https://github.com/krisukox/bazels3cache/tree/main/test/workspace).

Integration test can be run with:  
`make run-integration-test`

### Benchmark

Before running benchmark, please run submodule update to download the [Bazel](https://github.com/bazelbuild/bazel) repository:  
`git submodule update --init`

Benchmark builds the [Bazel](https://github.com/bazelbuild/bazel) project. It uses [netem](https://wiki.linuxfoundation.org/networking/netem) to simulate delay. Benchmark can be run with:  
`make run-benchmark`

Results of the benchmark will be available under the `test/results/` directory.

Benchmark configuration environment variables:  
- `DELAY` - value in milliseconds that natem uses to simulate delay.
- `BENCHMARK_TARGET` - available targets: `test_performance_1` and `test_performance_2`.
- `BAZEL_TARGET` - target of the bazel repository that the benchmark will build.

Default configuration is available [here](https://github.com/krisukox/bazels3cache/blob/main/test/benchmark.env).
