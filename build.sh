#!/bin/sh

set -e

VERSION=${VERSION:-$(./version.sh)}
PACKAGE=github.com/sol1du2/bouncer
DATE=${DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}

GO=${GO:-go}
GOLINT=${GOLINT:-golangci-lint}

${GO} mod vendor

LDFLAGS=${LDFLAGS:--s -w}
ASMFLAGS=${ASMFLAGS:-}
GCFLAGS=${GCFLAGS:-}

CGO_ENABLED=0 ${GO} build \
	-mod=vendor \
	-trimpath \
 	-tags release \
	-buildmode=exe \
	-asmflags "${ASMFLAGS}" \
	-gcflags "${GCFLAGS}" \
	-ldflags "${LDFLAGS} -buildid=reproducible/${VERSION} -X ${PACKAGE}/version.Version=${VERSION} -X ${PACKAGE}/version.BuildDate=${DATE} -extldflags -static" \
	-o bin/bouncerd ./cmd/bouncerd
