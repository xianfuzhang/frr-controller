import os
import sys
from jinja2 import FileSystemLoader, Environment

j2_loader = FileSystemLoader('./')

env = Environment(loader=j2_loader)

j2_tpl = env.get_template('./template.j2')

#shell传参
var_asn = os.getenv("ASN") or 0
var_neighbors = os.getenv("NEIGHBORS") or ""

mount_path = sys.argv[1]
if not os.path.exists(mount_path):
    os.makedirs(os.path.dirname(mount_path))

try:
    result = j2_tpl.render(ASN = var_asn, NEIGHBORS=var_neighbors.split(','))
except Exception as e:
    raise e

with open(mount_path, 'w') as f:
    f.write(result)

# print(result)
