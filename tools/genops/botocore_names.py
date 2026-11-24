# tools/genops/botocore_names.py - Print botocore's CLI name for every model operation as JSON
# Usage: python tools/genops/botocore_names.py > names.json
#
# The Go test compares its own name conversion against this output.

import gzip
import json
import os
import re

import botocore
from botocore import xform_name

# Walk the newest dated model of every service in the installed botocore
data = os.path.join(os.path.dirname(botocore.__file__), "data")
names = {}
for service in sorted(os.listdir(data)):
    directory = os.path.join(data, service)
    if not os.path.isdir(directory) or service.startswith("_"):
        continue
    versions = sorted(v for v in os.listdir(directory) if re.fullmatch(r"\d{4}-\d\d-\d\d", v))
    path = os.path.join(directory, versions[-1], "service-2.json.gz") if versions else ""
    if not os.path.exists(path):
        continue
    with gzip.open(path, "rt") as handle:
        for operation in json.load(handle)["operations"]:
            names[operation] = xform_name(operation, "-")
print(json.dumps(names))
