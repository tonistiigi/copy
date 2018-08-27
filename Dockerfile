# syntax = tonistiigi/dockerfile:runmount20180925

ARG BASEMODE=external

# xcross wraps go to automatically configure TARGETPLATFORM 
FROM --platform=$BUILDPLATFORM tonistiigi/xx:golang AS xcross

FROM --platform=$BUILDPLATFORM golang:1.11 AS main
RUN apt-get update && apt-get install -y file
COPY --from=xcross / /
WORKDIR /go/src/github.com/tonistiigi/copy
ENV CGO_ENABLED=0
ARG TARGETPLATFORM
RUN --mount=target=. --mount=target=/root/.cache,type=cache \
  go build -o /copy -ldflags '-s -w' github.com/tonistiigi/copy/cmd/copy && \
  file /copy | grep "statically linked"

FROM gruebel/upx AS upx
COPY --from=main /copy /copy
RUN ["upx", "/copy"]

FROM alpine AS wget
WORKDIR /out
RUN apk add --no-cache wget

# FROM wget AS cp
# RUN wget http://s.minos.io/archive/bifrost/x86_64/coreutils-7.6-5.tar.gz && \
#   echo "0b1e8bce191e8d15d3658836c07021c6806a840b1bb528dae744349598c0ad35a coreutils-7.6-5.tar.gz" | sha25sum -cs
# RUN tar xvf coreutils-7.6-5.tar.gz -C /

FROM wget AS tar
RUN wget http://s.minos.io/archive/bifrost/x86_64/tar-1.23-1.tar.gz && \
  echo "2794d1cd9eb1023eead80179fed13437a14874d2cb8170c54bac51a963e3a7bd  tar-1.23-1.tar.gz" | sha256sum -c
RUN tar xvf tar-1.23-1.tar.gz -C /out

FROM wget AS gz
RUN wget http://s.minos.io/archive/bifrost/x86_64/gzip-1.4-1.tar.bz2 && \
  echo "17b74107a011a2e5cefcdca3c87fe9985dcfb7459ad1784b2c43b7f2e9576ed6  gzip-1.4-1.tar.bz2" | sha256sum -c
RUN tar xvf gzip-1.4-1.tar.bz2 -C /out

FROM wget AS bz
RUN wget http://s.minos.io/archive/bifrost/x86_64/bzip2-bin-1.0.5-1.tar.gz && \
  echo "efb099eeda8208cbb77edc8fd380b3f680fb6106e92c2eb359263b45ce006d55  bzip2-bin-1.0.5-1.tar.gz" | sha256sum -c
RUN tar xvf bzip2-bin-1.0.5-1.tar.gz -C /out

FROM wget AS xz
RUN wget http://s.minos.io/archive/bifrost/x86_64/xz-5.0.3-1.tar.gz && \
  echo "1449e0b209169c559ed1f4c906d76bbcf27607c072484ab852523df86ee39450  xz-5.0.3-1.tar.gz" | sha256sum -c
RUN tar xvf xz-5.0.3-1.tar.gz  -C /out

# amd release stage is different from other arch and only has static binaries
FROM scratch AS release-amd64
COPY --from=upx /copy /bin/
COPY --from=tar /out/bin /bin/
COPY --from=gz /out/bin /bin/
COPY --from=bz /out/bin /bin/
COPY --from=xz /out/usr/bin /bin/

FROM alpine AS base-inline
RUN apk add --no-cache tar gzip bzip2 xz

FROM tonistiigi/copy:$TARGETARCH$TARGETVARIANT-base AS base-external

# base image for non-amd can be switch between external and inlined
FROM base-$BASEMODE AS release-noamd64
COPY --from=main /copy /bin/

FROM release-noamd64 AS release-arm
FROM release-noamd64 AS release-arm64
FROM release-noamd64 AS release-s390x
FROM release-noamd64 AS release-ppc64le

# main release stage
FROM release-$TARGETARCH AS release
ENTRYPOINT ["/bin/copy"]

# dev image stage for debugging
FROM alpine AS dev-env
COPY --from=release /bin/ /bin/
ENTRYPOINT ["ash"]

# set default back to release
FROM release
