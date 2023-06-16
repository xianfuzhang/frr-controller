FROM alpine:3.17
RUN apk add \
		--no-cache \
		--update-cache \
		openvswitch 
COPY init-network.sh / 
# Simple init manager for reaping processes and forwarding signals
ENTRYPOINT ["/sbin/tini", "--"]

# Default CMD starts watchfrr
CMD ["/init-network.sh"]