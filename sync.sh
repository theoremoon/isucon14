#!/bin/bash

set -eux

rsync -az --exclude-from=".gitignore" --exclude=".git" -e "ssh" ./ isucon1:/home/isucon

# scp config-alp.yaml isucon1:config-alp.yaml
scp log.sh isucon1:log.sh
ssh isucon1 "chmod +x log.sh reload.sh"
# scp env.sh isucon1:env.sh
# scp ./google-cred.json isucon:google-cred.json
