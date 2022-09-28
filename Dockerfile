FROM golang:1.18-alpine as builder
COPY . /go/src/github.com/storageos/kubectl-storageos
WORKDIR /go/src/github.com/storageos/kubectl-storageos
RUN apk add curl make bash
# Install latest kubectl from source
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
ENV CGO_ENABLED=0
# Ideally would use make build, but was failing with getting glibc version issues from controller-gen
RUN make _build

FROM alpine
COPY --from=builder /go/src/github.com/storageos/kubectl-storageos/bin/kubectl-storageos /kubectl-storageos
# Add the binaries to the path
RUN ln -s /kubectl-storageos /usr/local/bin/kubectl-storageos
COPY --from=builder /usr/local/bin/kubectl /usr/local/bin/kubectl
ENTRYPOINT ["/kubectl-storageos"]
