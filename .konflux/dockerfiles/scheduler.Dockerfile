ARG GO_BUILDER=registry.access.redhat.com/ubi9/go-toolset:1.25
ARG RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:b9b10f42d7eba7ad4a6d5ef26b7d34fdc892b2ffe59b8d0372ec884008569eb6

FROM $GO_BUILDER AS builder

WORKDIR /go/src/github.com/tektoncd/tekton-kueue
COPY upstream .
COPY .konflux/patches patches/
RUN set -e; for f in patches/*.patch; do echo ${f}; [[ -f ${f} ]] || continue; git apply ${f}; done
COPY head HEAD


ENV GODEBUG="http2server=0"
RUN go mod download
RUN go build -tags disable_gcp -ldflags="-X 'knative.dev/pkg/changeset.rev=${CHANGESET_REV:0:7}'" -o /tmp/tekton-kueue \
    ./cmd/main.go
# RUN /bin/sh -c 'echo $CI_OPERATOR_UPSTREAM_COMMIT > /tmp/HEAD'

FROM $RUNTIME

ENV KUEUE=/tmp/tekton-kueue  \
    KO_DATA_PATH=/kodata

COPY --from=builder $KUEUE $KUEUE

LABEL \
    com.redhat.component="openshift-pipelines-scheduler-rhel9-container" \
    cpe="cpe:/a:redhat:openshift_pipelines:1.24::el9" \
    description="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.k8s.description="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.k8s.display-name="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.openshift.tags="tekton,openshift,tekton-kueue,scheduler" \
    maintainer="pipelines-extcomm@redhat.com" \
    name="openshift-pipelines/pipelines-scheduler-rhel9" \
    summary="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    version="v1.24.0"

RUN groupadd -r -g 65532 nonroot && useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT [ "${KUEUE}" ]
