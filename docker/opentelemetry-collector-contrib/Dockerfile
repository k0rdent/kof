FROM otel/opentelemetry-collector-contrib:0.123.0 AS orig

FROM linuxcontainers/debian-slim:latest
RUN apt-get -y update && apt-get -y install systemd ca-certificates && apt-get clean && rm -rf /var/lib/apt/lists/*
ARG USER_UID=10001
ARG USER_GID=10001
RUN groupadd --gid ${USER_GID} oteluser && \
    useradd --uid ${USER_UID} --gid ${USER_GID} oteluser && \
    usermod -aG systemd-journal oteluser
USER ${USER_UID}:${USER_GID}
COPY --from=orig /otelcol-contrib /
COPY --from=orig /etc/otelcol-contrib/config.yaml /etc/otelcol-contrib/config.yaml
EXPOSE 4317 55680 55679
ENTRYPOINT ["/otelcol-contrib"]
CMD ["--config", "/etc/otelcol-contrib/config.yaml"]
