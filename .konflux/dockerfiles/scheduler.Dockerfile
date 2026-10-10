ARG GO_BUILDER=registry.access.redhat.com/ubi9/go-toolset:latest@sha256:6f246e8913d082df463b62a74c72f0d2b410583e1b2ac48add39cd7ede59ce62
ARG UBI_RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal@sha256:5ed244b62bbf4095080144d9d35eb8fcd3d39a9801f94aadd63b9d10978a01ae

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
    cpe="cpe:/a:redhat:openshift_pipelines:1.24::el9" \
    description="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.k8s.description="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.k8s.display-name="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    io.openshift.tags="tekton,openshift,tekton-kueue,scheduler" \
    maintainer="pipelines-extcomm@redhat.com" \
    name="openshift-pipelines/pipelines-scheduler-rhel9" \
    summary="Red Hat OpenShift Pipelines tekton-kueue scheduler" \
    version="v1.24.2"

RUN groupadd -r -g 65532 nonroot && useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT [ "/manager" ]