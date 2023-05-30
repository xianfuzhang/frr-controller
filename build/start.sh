#!/bin/sh

mount_path="/etc/frr/frr.conf"
if [ $# -gt 0 ]; then
    mount_path=$1
fi

/usr/bin/python3 render.py ${mount_path}

# /bin/sh "-c" "tail -f /dev/null"