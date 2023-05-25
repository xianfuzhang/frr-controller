FROM alpine:3.14

RUN apk add --no-cache python3 py3-pip

RUN pip3 install pyyaml jinja2

WORKDIR /workspace

COPY ./build/. .

RUN chmod +x render.py  \
    && /usr/bin/python3 render.py

CMD ["/bin/sh", "-c", "tail -f /dev/null"]