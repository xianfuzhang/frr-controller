FROM fedora:34

USER root

ENV PYTHONDONTWRITEBYTECODE yes

RUN INSTALL_PKGS=" \
        python3-pyyaml bind-utils procps-ng openssl numactl-libs firewalld-filesystem \
        libpcap hostname kubernetes-client tcpdump  python3-pyOpenSSL \
        iptables iproute iputils strace socat \
        " && \
        dnf install --best --refresh -y --setopt=tsflags=nodocs $INSTALL_PKGS && \
        dnf clean all && rm -rf /var/cache/dnf/*

COPY frr-controller /usr/bin
COPY frr.sh /root/
COPY git_info /root

LABEL io.k8s.display-name="frr-controller" \
      io.k8s.description="This is a Kubernetes controller that provides a frr controller."

WORKDIR /root
ENTRYPOINT /root/frr.sh
