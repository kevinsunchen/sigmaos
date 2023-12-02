#!/bin/bash

usage() {
  echo "Usage: $0 --vpc VPC [--n N] [--taint N:M]" 1>&2
}

VPC=""
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
  --vpc)
    shift
    VPC=$1
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

if [ -z "$VPC" ] || [ $# -gt 0 ]; then
    usage
    exit 1
fi

vms=`./lsvpc.py $VPC | grep -w VMInstance | cut -d " " -f 5`
vms_privaddr=`./lsvpc.py $VPC --privaddr | grep -w VMInstance | cut -d " " -f 6`

vma=($vms)
vma_privaddr=($vms_privaddr)
MAIN="${vma[0]}"
MAIN_PRIVADDR="${vma_privaddr[0]}"

for vm in $vms; do
  echo "VM: $vm"
  # No additional benchmarking setup needed for AWS.
  ssh -i key-$VPC.pem ubuntu@$vm /bin/bash <<ENDSSH
    cd sigmaos
    git fetch --all
    git checkout osdi23-submit
    git pull
#    ./make.sh --norace --version RETRY
    ./install.sh --realm test-realm --version RETRY
ENDSSH
done
