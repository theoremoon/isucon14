#!/bin/bash

NOW=$(date "+%Y%m%d%H%M%S")
sudo cat /var/log/nginx/access.log | alp ltsv --config config-alp.yaml | sudo tee /srv/html/alp/${NOW}.html > /dev/null

sudo cat /var/log/mysql/mysql-slow.log | pt-query-digest | sudo tee /srv/html/pt-query-digest/${NOW}.txt > /dev/null
