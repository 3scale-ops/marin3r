FROM registry.access.redhat.com/ubi8/ubi-minimal
ARG RELEASE

COPY build/marin3r_amd64_${RELEASE} /marin3r

EXPOSE 8443
EXPOSE 18000
ENTRYPOINT [ "/marin3r" ]