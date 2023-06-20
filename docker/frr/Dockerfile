FROM quay.io/frrouting/frr:8.5.1
RUN apk add \
		--no-cache \
		--update-cache \
		openvswitch 
RUN sed -i -e '/bgpd/s/no/yes/' /etc/frr/daemons

# Simple init manager for reaping processes and forwarding signals
ENTRYPOINT ["/sbin/tini", "--"]

# Default CMD starts watchfrr
CMD ["/usr/lib/frr/docker-start"]