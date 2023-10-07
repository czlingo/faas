FROM ubuntu:20.04 AS gobuilder

RUN apt update && apt upgrade -y
RUN apt install wget -y

WORKDIR /download
ARG GO_VERSION=go1.20.11.linux-amd64
RUN wget https://go.dev/dl/$GO_VERSION.tar.gz && tar -xf $GO_VERSION.tar.gz

WORKDIR /go
RUN mv /download/go .

FROM ubuntu:20.04

ENV CNB_USER_ID=1000
ENV CNB_GROUP_ID=1000
ENV CNB_STACK_ID="faas.czlingo.buildpacks.stacks"
LABEL io.buildpacks.stack.id="faas.czlingo.buildpacks.stacks"

RUN groupadd cnb --gid ${CNB_GROUP_ID} && \
  useradd --uid ${CNB_USER_ID} --gid ${CNB_GROUP_ID} -m -s /bin/bash cnb

WORKDIR /go
COPY --from=gobuilder /go/go .
ENV PATH=$PATH:/go/bin

WORKDIR /

USER ${CNB_USER_ID}:${CNB_GROUP_ID}