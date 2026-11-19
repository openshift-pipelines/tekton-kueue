ARG GO_BUILDER=registry.redhat.io/ubi9/go-toolset:latest@sha256:51eef2836e5223c41577757c30544fbfb3f0710f3436b70b82d2195e35fcb743
ARG UBI_RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:d8162789d7913ef8941481687b92ab4bbec311870948f66cf52b9cc7e0c57c06

FROM $GO_BUILDER AS builder

WORKDIR /go/src/github.com/tektoncd/tekton-kueue
COPY upstream .
COPY .konflux/patches patches/
RUN set -e; for f in patches/*.patch; do echo ${f}; [[ -f ${f} ]] || continue; git apply ${f}; done
COPY head HEAD


ENV GODEBUG="http2server=0"
RUN go build -tags disable_gcp -ldflags="-X 'knative.dev/pkg/changeset.rev=${CHANGESET_REV:0:7}'" -o /tmp/manager \
    ./cmd/main.go
# RUN /bin/sh -c 'echo $CI_OPERATOR_UPSTREAM_COMMIT > /tmp/HEAD'

FROM $UBI_RUNTIME

COPY --from=builder /tmp/manager /manager
LABEL \
    com.redhat.component="openshift-pipelines-scheduler-rhel9-container" \
    cpe="cpe:/a:redhat:openshift_pipelines:1.23::el9" \
    description="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.k8s.description="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.k8s.display-name="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.openshift.tags="tekton,openshift,tekton-kueue,scheduler" \
    maintainer="pipelines-extcomm@redhat.com" \
    name="openshift-pipelines/pipelines-scheduler-rhel9" \
    summary="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    version="v1.23.5"

RUN groupadd -r -g 65532 nonroot && useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT [ "/manager" ]