# --- build smart gateway and agent plugins ---
FROM registry.access.redhat.com/ubi8 AS builder
ENV GOPATH=/go
ENV CORE=/go/src/github.com/infrawatch/sg-core
ENV WD=/go/src/github.com/infrawatch/sg-agent

WORKDIR $WD
COPY . $WD/

# dependencies for qpid-proton-c
COPY build/repos/opstools.repo /etc/yum.repos.d/opstools.repo
RUN dnf install golang git qpid-proton-c-devel -y --setopt=tsflags=nodocs

# build sg-core
RUN git clone https://github.com/infrawatch/sg-core $CORE && pushd $CORE && PRODUCTION_BUILD=true ./build.sh && popd

# sync go.mod with local sg-core
RUN sed -i "s/go [0-9\.]\{1,\}/$(grep -e '^go [0-9\.]' ../sg-core/go.mod)/" go.mod
RUN echo "replace $(grep sg-core go.mod | tr '\n' ' ' | tr '\t' ' ')=> $CORE" >> go.mod

# build agent plugins
RUN ./build.sh

# --- end build, create smart gateway layer ---

FROM registry.access.redhat.com/ubi8-minimal

LABEL io.k8s.display-name="Smart Gateway Agent" \
      io.k8s.description="A component of the Service Telemetry Framework on the client side that provides closed loop remediation" \
      maintainer="Martin Mágr <mmagr@redhat.com>"

COPY --from=builder /tmp/sg-core /
COPY --from=builder /tmp/plugins/*.so /usr/lib64/sg-core/

ENTRYPOINT ["/sg-core"]
