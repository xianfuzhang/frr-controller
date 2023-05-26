#!/bin/sh

/usr/bin/python3 render.py
/bin/sh "-c" "tail -f /dev/null"