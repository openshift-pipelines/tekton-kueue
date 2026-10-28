ARG GO_BUILDER=registry.redhat.io/ubi9/go-toolset:latest@sha256:edb9452603fa48e9882063f001ecd7d322add20f1564a4f05f38f6683be5ea94
ARG UBI_RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal@sha256:2d4c4551af6831dcd26bbf86108c4385de5183ec8fe6b5868f6f9640fe47f4ed

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
    com.redhat.component="openshift-pipelines-kueue-rhel9-container" \
    cpe="cpe:/a:redhat:openshift_pipelines:1.24::el9" \
    description="Red Hat OpenShift Pipelines tekton-kueue kueue" \
    io.k8s.description="Red Hat OpenShift Pipelines tekton-kueue kueue" \
    io.k8s.display-name="Red Hat OpenShift Pipelines tekton-kueue kueue" \
    io.openshift.tags="tekton,openshift,tekton-kueue,kueue" \
    maintainer="pipelines-extcomm@redhat.com" \
    name="openshift-pipelines/pipelines-kueue-rhel9" \
    summary="Red Hat OpenShift Pipelines tekton-kueue kueue" \
    version="v1.24.2"

RUN groupadd -r -g 65532 nonroot && useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532

ENTRYPOINT [ "/manager" ]
