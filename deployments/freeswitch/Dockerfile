FROM debian:bullseye-slim

# Install FreeSWITCH from official repository
RUN apt-get update && apt-get install -y \
    gnupg2 \
    wget \
    lsb-release \
    ca-certificates \
    && wget -O - https://files.freeswitch.org/repo/deb/debian-release/fsstretch-archive-keyring.asc | apt-key add - \
    && echo "deb http://files.freeswitch.org/repo/deb/debian-release/ bullseye main" > /etc/apt/sources.list.d/freeswitch.list \
    && apt-get update \
    && apt-get install -y \
        freeswitch \
        freeswitch-mod-commands \
        freeswitch-mod-conference \
        freeswitch-mod-console \
        freeswitch-mod-dialplan-xml \
        freeswitch-mod-dptools \
        freeswitch-mod-event-socket \
        freeswitch-mod-hash \
        freeswitch-mod-loopback \
        freeswitch-mod-opus \
        freeswitch-mod-sndfile \
        freeswitch-mod-sofia \
        freeswitch-mod-tone-stream \
        freeswitch-mod-verto \
        freeswitch-sounds-en-us-callie \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create directories
RUN mkdir -p /recordings /var/log/freeswitch

# Copy configuration files
COPY conf/ /etc/freeswitch/

# Set permissions
RUN chown -R freeswitch:freeswitch /etc/freeswitch /recordings /var/log/freeswitch

# Expose ports
EXPOSE 5060/udp 5060/tcp
EXPOSE 8021/tcp
EXPOSE 8081/tcp
EXPOSE 16384-16484/udp

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD fs_cli -x "status" || exit 1

USER freeswitch

ENTRYPOINT ["/usr/bin/freeswitch", "-nonat", "-nf"]
