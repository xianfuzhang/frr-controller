import os
import yaml
from jinja2 import FileSystemLoader, Environment

j2_loader = FileSystemLoader('./')

env = Environment(loader=j2_loader)

j2_tpl = env.get_template('./template.j2')

#shell传参
# var_asn = os.getenv("ASN")
# var_neighbors = os.getenv("NEIGHBORS")

#jsonfile传参
with open('./variables.json', closefd=True) as f:
    data = yaml.safe_load(f)

try:
    # result = j2_tpl.render(asn = var_asn, neighbors=var_neighbors)
    result = j2_tpl.render(data)
except Exception as e:
    raise e

with open('./frr.conf', 'w') as f:
    f.write(result)

# print(result)
