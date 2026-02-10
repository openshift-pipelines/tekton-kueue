ARG GO_BUILDER=registry.access.redhat.com/ubi9/go-toolset:9.7-1770596585
ARG RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:ecd4751c45e076b4e1e8d37ac0b1b9c7271930c094d1bcc5e6a4d6954c6b2289

FROM $GO_BUILDER AS builder

WORKDIR /go/src/github.com/tektoncd/tekton-scheduler
COPY upstream .
COPY .konflux/patches patches/
RUN set -e; for f in patches/*.patch; do echo ${f}; [[ -f ${f} ]] || continue; git apply ${f}; done
COPY head HEAD


ENV GODEBUG="http2server=0"
RUN go build -tags disable_gcp -ldflags="-X 'knative.dev/pkg/changeset.rev=${CHANGESET_REV:0:7}'" -o /tmp/manager \
    ./cmd/main.go
# RUN /bin/sh -c 'echo $CI_OPERATOR_UPSTREAM_COMMIT > /tmp/HEAD'

FROM $RUNTIME

ENV KUEUE=/tmp/manager

COPY --from=builder $KUEUE $KUEUE
ARG VERSION=1.22.0
LABEL \
      com.redhat.component="openshift-pipelines-rhel9-scheduler" \
      name="openshift-pipelines/pipelines-rhel9-scheduler" \
      version="$VERSION" \
      summary="Red Hat OpenShift Pipelines Scheduler" \
      maintainer="pipelines-extcomm@redhat.com" \
      description="Red Hat OpenShift Pipelines Scheduler" \
      io.k8s.display-name="Red Hat OpenShift Pipelines Scheduler" \
      io.k8s.description="Red Hat OpenShift Pipelines Scheduler" \
      io.openshift.tags="scheduler,tekton,openshift"

RUN groupadd -r -g 65532 nonroot && useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT [ "${KUEUE}" ]
