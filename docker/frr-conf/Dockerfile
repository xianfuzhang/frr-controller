FROM alpine:3.14

RUN apk add --no-cache python3 py3-pip  \
    && pip3 install pyyaml jinja2

# WORKDIR /workspace
COPY render.py ./
COPY start.sh ./
COPY template.j2 ./

RUN chmod +x render.py start.sh

ENTRYPOINT ["./start.sh"]