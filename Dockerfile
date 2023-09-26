FROM ubuntu:22.04
COPY sentinel_tunnel /usr/local/bin/
COPY entrypoint /usr/local/bin
RUN mkdir /etc/sentinel_tunnel && \
    chown www-data /etc/sentinel_tunnel && \
    chmod g+rwx /etc/sentinel_tunnel && \
    adduser www-data root && \
    apt update && \
    apt install --assume-yes --no-install-recommends redis-tools && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["/usr/local/bin/entrypoint"]
CMD ["/usr/local/bin/sentinel_tunnel", "/etc/sentinel_tunnel/config.json", "/dev/stdout"]
USER www-data
