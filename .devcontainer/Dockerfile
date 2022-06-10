# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.202.5/containers/go/.devcontainer/base.Dockerfile

# [Choice] Go version (use -bullseye variants on local arm64/Apple Silicon): 1, 1.16, 1.17, 1-bullseye, 1.16-bullseye, 1.17-bullseye, 1-buster, 1.16-buster, 1.17-buster
ARG VARIANT="1.18"
FROM mcr.microsoft.com/vscode/devcontainers/go:0-${VARIANT}

# [Optional] Uncomment this section to install additional OS packages.
# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y install --no-install-recommends <your-package-list-here>

# [Optional] Uncomment the next line to use go get to install anything else you need
# RUN go get -x <your-dependency-or-tool>

ARG KIND="0.11.1"
RUN curl -Lo /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v${KIND}/kind-linux-amd64 && \
    chmod +x /usr/local/bin/kind

ARG KUTTL="0.11.1"
RUN curl -Lo /usr/local/bin/kubectl-kuttl https://github.com/kudobuilder/kuttl/releases/download/v${KUTTL}/kubectl-kuttl_${KUTTL}_linux_x86_64 && \
    chmod +x /usr/local/bin/kubectl-kuttl