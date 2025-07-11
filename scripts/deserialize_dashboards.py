import yaml
import sys
import os

dest_dir = sys.argv[1]

all = yaml.safe_load_all(sys.stdin.read())
for d in all:
  with open(f"{dest_dir}/{d['key']}", "w") as f:
     f.write(d['value'])
