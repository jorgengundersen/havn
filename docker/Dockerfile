FROM ubuntu:24.04

ARG UID=1000
ARG GID=1000

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    coreutils \
    curl \
    nix-bin \
    openssh-server \
    sudo \
    tini \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --gid "${GID}" devuser \
    && useradd --uid "${UID}" --gid "${GID}" --create-home --shell /bin/bash devuser

RUN install -d -m 0755 /run/sshd /nix \
    && install -d -m 0700 -o devuser -g devuser /home/devuser/.ssh \
    && install -d -m 0755 -o devuser -g devuser /home/devuser/.local/share /home/devuser/.cache /home/devuser/.local/state

RUN printf 'devuser ALL=(root) NOPASSWD: /usr/sbin/sshd\n' >/etc/sudoers.d/devuser-sshd \
    && chmod 0440 /etc/sudoers.d/devuser-sshd

RUN mkdir -p /etc/ssh/sshd_config.d \
    && printf 'PasswordAuthentication no\nPubkeyAuthentication yes\nChallengeResponseAuthentication no\nUsePAM yes\n' >/etc/ssh/sshd_config.d/havn.conf
