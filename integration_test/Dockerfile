FROM golang:1.20.0-bullseye

# Install bazel
RUN apt-get update && \
    apt-get install apt-transport-https curl gnupg -y && \
    curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor >bazel-archive-keyring.gpg && \
    mv bazel-archive-keyring.gpg /usr/share/keyrings && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/bazel-archive-keyring.gpg] https://storage.googleapis.com/bazel-apt stable jdk1.8" | tee /etc/apt/sources.list.d/bazel.list && \
    apt-get update && \
    apt-get install -y bazel-6.0.0 patch python3 iputils-ping iproute2 && \
    ln -s /usr/bin/bazel-6.0.0 /usr/bin/bazel

ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.9.0/wait /wait
RUN chmod +x /wait

COPY integration_test/workspace /integration_test/workspace
COPY integration_test/.aws/* /integration_test/.aws/
RUN cp -r /integration_test/.aws/ ~/ && \
    cat ~/.aws/credentials

# Download golang dependencies
COPY go.* Makefile /src/
WORKDIR /src
RUN go mod download

# Build bazels3cache app
COPY app/*.go /src/app/
RUN make build-debug

ENV test_workspace /integration_test/workspace
ENV bazels3cache ./bazels3cache

COPY integration_test/test.py /integration_test/
CMD /wait && python3 -u /integration_test/test.py
