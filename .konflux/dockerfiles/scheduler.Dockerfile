ARG GO_BUILDER=registry.access.redhat.com/ubi9/go-toolset:latest
ARG RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:5b74fce9d6e629942a0c6dc0f546c193e70d7f974d999a48c948c53dd3d36362

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

ARG VERSION=1.22

COPY --from=builder /tmp/manager /manager
LABEL \
    com.redhat.component="openshift-pipelines-scheduler-rhel9-container" \
    cpe="cpe:/a:redhat:openshift_pipelines:1.22::el9" \
    description="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.k8s.description="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.k8s.display-name="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.openshift.tags="tekton,openshift,tekton-kueue,scheduler" \
    maintainer="pipelines-extcomm@redhat.com" \
    name="openshift-pipelines/pipelines-scheduler-rhel9" \
    summary="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    version="v1.22.2"

RUN groupadd -r -g 65532 nonroot && useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT [ "/manager" ]
