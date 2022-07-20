#!/bin/bash

# Adds a new instance to an existing VPC, combining mkvpc.py and
# setup-instance.sh.

usage() {
  echo "Usage: $0 --vpc VPC --vm VM-name" 1>&2
}

VPC=""
NAME=""
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
  --vpc)
    shift
    VPC=$1
    shift
    ;;
  --vm)
    shift
    NAME=$1
    shift
    ;;
  -help)
    usage
    exit 0
    ;;
  *)
    echo "Error: unexpected argument '$1'"
    usage
    exit 1
    ;;
  esac
done

if [ -z "$VPC" ] || [ -z "$NAME" ] || [ $# -gt 0 ]; then
    usage
    exit 1
fi

./mkvpc.py --vpc $VPC $NAME

vm=`./lsvpc.py $VPC | grep -w $NAME | cut -d " " -f 5`
echo "SETUP $vm"
./setup-instance.sh --vpc $VPC --vm $vm



