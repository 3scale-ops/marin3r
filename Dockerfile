FROM debian:buster-slim
ARG RELEASE

COPY build/marin3r_amd64_${RELEASE} /marin3r

EXPOSE 8080
ENTRYPOINT [ "/marin3r" ]