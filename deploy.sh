#!/bin/bash

set -eux

source ./sync.sh
ssh isucon1 "bash ./reload.sh"
