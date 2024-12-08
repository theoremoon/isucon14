#!/bin/bash

set -eux

(cd /home/isucon/webapp/go; /home/isucon/local/golang/bin/go build -o isuride)
sudo systemctl restart isuride-go.service

# nginx
# sudo rm -rf /var/log/nginx/access.log
# sudo mkdir -p /srv/html/alp
sudo rsync --chown root:root -avz /home/isucon/nginx/isuride.conf /etc/nginx/sites-available/isuride.conf
sudo systemctl restart nginx

# mysql
# sudo rm -rf /var/log/mysql/mysql-slow.log
# sudo mkdir -p /srv/html/pt-query-digest
# sudo rsync --chown root:root -avz /home/isucon/mysql.conf.d /etc/mysql/
# sudo systemctl restart mysql
