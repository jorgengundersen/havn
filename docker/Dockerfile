FROM ubuntu:24.04

ARG UID=1000
ARG GID=1000

ENV DEBIAN_FRONTEND=noninteractive

RUN sed -i 's|http://archive.ubuntu.com/ubuntu|https://archive.ubuntu.com/ubuntu|g; s|http://security.ubuntu.com/ubuntu|https://security.ubuntu.com/ubuntu|g' /etc/apt/sources.list.d/ubuntu.sources \
    && apt-get update && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    coreutils \
    curl \
    nix-bin \
    openssh-server \
    sudo \
    tini \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc/nix \
    && printf 'experimental-features = nix-command flakes\n' >/etc/nix/nix.conf

RUN if ! getent group "${GID}" >/dev/null 2>&1; then \
      if getent group devuser >/dev/null 2>&1; then \
        groupmod --gid "${GID}" devuser; \
      else \
        groupadd --gid "${GID}" devuser; \
      fi; \
    else \
      existing_gid_group="$(getent group "${GID}" | cut -d: -f1)"; \
      if [ "${existing_gid_group}" != "devuser" ] && ! getent group devuser >/dev/null 2>&1; then \
        groupmod -n devuser "${existing_gid_group}"; \
      fi; \
    fi \
    && existing_uid_user="$(getent passwd "${UID}" | cut -d: -f1 || true)" \
    && if id -u devuser >/dev/null 2>&1; then \
         usermod --uid "${UID}" --gid "${GID}" --shell /bin/bash devuser; \
         usermod --home /home/devuser devuser; \
       elif [ -n "${existing_uid_user}" ]; then \
         if [ "${existing_uid_user}" != "devuser" ]; then \
           usermod -l devuser --gid "${GID}" --shell /bin/bash "${existing_uid_user}"; \
         fi; \
         usermod --home /home/devuser devuser; \
       else \
         useradd --uid "${UID}" --gid "${GID}" --shell /bin/bash --create-home devuser; \
       fi \
    && install -d -m 0755 -o devuser -g "${GID}" /home/devuser

RUN install -d -m 0755 /run/sshd \
    && install -d -m 0755 -o devuser -g "${GID}" /nix /nix/var /nix/var/nix \
    && install -d -m 0700 -o devuser -g "${GID}" /home/devuser/.ssh \
    && install -d -m 0755 -o devuser -g "${GID}" /home/devuser/.local/share /home/devuser/.cache /home/devuser/.local/state

RUN printf 'devuser ALL=(root) NOPASSWD: /usr/sbin/sshd\n' >/etc/sudoers.d/devuser-sshd \
    && chmod 0440 /etc/sudoers.d/devuser-sshd

RUN mkdir -p /etc/ssh/sshd_config.d \
    && printf 'PasswordAuthentication no\nPubkeyAuthentication yes\nChallengeResponseAuthentication no\nUsePAM yes\n' >/etc/ssh/sshd_config.d/havn.conf
