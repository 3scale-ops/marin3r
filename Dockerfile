FROM debian:buster-slim
ARG release

RUN apt update && apt -y install ca-certificates
COPY build/marin3r_amd64_${release} /marin3r

EXPOSE 8080
ENTRYPOINT [ "/marin3r" ]