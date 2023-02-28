FROM golang:1.20.0-bullseye AS base

# Install bazel
RUN apt-get update && \
    apt-get install apt-transport-https curl gnupg git -y && \
    curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor >bazel-archive-keyring.gpg && \
    mv bazel-archive-keyring.gpg /usr/share/keyrings && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/bazel-archive-keyring.gpg] https://storage.googleapis.com/bazel-apt stable jdk1.8" | tee /etc/apt/sources.list.d/bazel.list && \
    apt-get update && \
    apt-get install -y bazel-6.0.0 patch python3 iputils-ping iproute2 && \
    ln -s /usr/bin/bazel-6.0.0 /usr/bin/bazel

# Additional packages needed for benchmark
RUN apt-get install -y python3-pip default-jdk zip \
    && pip install requests

ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.9.0/wait /wait
RUN chmod +x /wait

COPY test/workspace /test/workspace
COPY test/.aws/* /test/.aws/
RUN cp -r /test/.aws/ ~/ && \
    cat ~/.aws/credentials

# Download golang dependencies
COPY go.* Makefile /src/
WORKDIR /src
RUN go mod download

# Build bazels3cache app
COPY app/*.go /src/app/
RUN make build-debug && cp ./bazels3cache /bin/bazels3cache

ENV TEST_WORKSPACE /test/workspace
ENV BAZELS3CACHE /bin/bazels3cache

COPY test/test.py /test/
CMD /wait && python3 -u /test/test.py ${TEST_TARGET}
