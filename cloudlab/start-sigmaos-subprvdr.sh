#!/bin/bash

usage() {
  echo "Usage: $0 [--vpc VPC] [--branch BRANCH] [--reserveMcpu rmcpu] [--pull TAG] [--n N_VM] [--ncores NCORES] [--overlays] [--turbo]" 1>&2
}

VPC=""
N_VM=""
NCORES=4
UPDATE=""
TAG=""
OVERLAYS=""
PROVIDER="cloudlab"
TOKEN=""
TURBO=""
RMCPU="0"
BRANCH="master"
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
  --vpc)
    shift
    VPC=$1
    shift
    ;;
  --n)
    shift
    N_VM=$1
    shift
    ;;
  --branch)
    shift
    BRANCH=$1
    shift
    ;;
  --ncores)
    shift
    NCORES=$1
    shift
    ;;
  --turbo)
    shift
    TURBO="--turbo"
    ;;
  --pull)
    shift
    TAG=$1
    shift
    ;;
  --overlays)
    shift
    OVERLAYS="--overlays"
    ;;
  --reserveMcpu)
    shift
    RMCPU="$1"
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

if [ $# -gt 0 ]; then
    usage
    exit 1
fi

if [ $NCORES -ne 4 ] && [ $NCORES -ne 2 ] && [ $NCORES -ne 20 ] && [ $NCORES -ne 40 ]; then
  echo "Bad ncores $NCORES"
  exit 1
fi

DIR=$(dirname $0)
source $DIR/env.sh

vms=`cat servers.txt | cut -d " " -f2`

aws_vms_full=$(../aws/lsvpc-alt.py  --privaddr $VPC)
aws_vms=`echo "$aws_vms_full" | grep -w VMInstance | cut -d " " -f 7`

vma=($vms)
aws_vma=($aws_vms)

MAIN="${vma[0]}"
MAIN_PRIVADDR=$(./leader-ip.sh)
MAIN_PUBADDR="${aws_vma[0]}"
SIGMASTART=$MAIN
# SIGMASTART_PRIVADDR=$MAIN_PRIVADDR
SIGMASTART_PUBADDR=$MAIN_PUBADDR
echo "SIGMASTART_PUBADDR $SIGMASTART_PUBADDR"
IMGS="arielszekely/sigmauser arielszekely/sigmaos arielszekely/sigmaosbase"

if ! [ -z "$N_VM" ]; then
  vms=${vma[@]:0:$N_VM}
fi

if ! [ -z "$TAG" ]; then
  ./update-repo.sh --parallel --branch $BRANCH
fi

vm_ncores=$(ssh -i $DIR/keys/cloudlab-sigmaos $LOGIN@$MAIN nproc)

for vm in $vms; do
  echo "starting SigmaOS on $vm!"
  $DIR/setup-for-benchmarking.sh $vm $TURBO
  # Get hostname.
  VM_NAME=$(ssh -i $DIR/keys/cloudlab-sigmaos $LOGIN@$vm hostname -s)
  KERNELID="sigma-$VM_NAME-$(echo $RANDOM | md5sum | head -c 3)"
  ssh -i $DIR/keys/cloudlab-sigmaos $LOGIN@$vm <<ENDSSH
  mkdir -p /tmp/sigmaos
  export SIGMADEBUG="$SIGMADEBUG"
  if [ $NCORES -eq 2 ]; then
    ./sigmaos/set-cores.sh --set 0 --start 2 --end $vm_ncores > /dev/null
    echo "ncores:"
    nproc
  else
    if [ $NCORES -eq 4 ]; then
      ./sigmaos/set-cores.sh --set 1 --start 2 --end 3 > /dev/null
      ./sigmaos/set-cores.sh --set 0 --start 4 --end 39 > /dev/null
      echo "ncores:"
      nproc
    else
      if [ $NCORES -eq 20 ]; then
        ./sigmaos/set-cores.sh --set 0 --start 20 --end 39 > /dev/null
        ./sigmaos/set-cores.sh --set 1 --start 2 --end 19 > /dev/null
        echo "ncores:"
        nproc
      else
        if [ $NCORES -eq 40 ]; then
          ./sigmaos/set-cores.sh --set 1 --start 2 --end 39 > /dev/null
          echo "ncores:"
          nproc
        fi
      fi
    fi
  fi

  aws s3 --profile sigmaos cp s3://kschen-9ps3/img-src/1.jpg ~/
  aws s3 --profile sigmaos cp s3://kschen-9ps3/img-src/6.jpg ~/
  aws s3 --profile sigmaos cp s3://kschen-9ps3/img-src/7.jpg ~/
  # aws s3 --profile sigmaos cp s3://kschen-9ps3/img-src/8.jpg ~/

  cd sigmaos

  echo "$PWD $SIGMADEBUG"
  if [ "${vm}" = "${MAIN}" ]; then
    echo "START DB: ${MAIN_PRIVADDR}"
    ./start-db.sh
    ./make.sh --norace linux
    echo "START NETWORK $MAIN_PRIVADDR"
    ./start-network.sh --addr $MAIN_PRIVADDR
    echo "START ${SIGMASTART} ${SIGMASTART_PUBADDR} ${KERNELID}"
    # if ! docker ps | grep -q etcd ; then
    #   # echo "START etcd"
    #   # ./start-etcd.sh
    # fi
    ./start-kernel.sh --boot lcschednode --named ${SIGMASTART_PUBADDR} --pull ${TAG} --reserveMcpu ${RMCPU} --dbip ${MAIN_PRIVADDR}:4406 --mongoip ${MAIN_PRIVADDR}:4407 ${OVERLAYS} --provider ${PROVIDER} ${KERNELID} 2>&1 | tee /tmp/start.out
    docker cp ~/1.jpg ${KERNELID}:/home/sigmaos/1.jpg
    docker cp ~/6.jpg ${KERNELID}:/home/sigmaos/6.jpg
    docker cp ~/7.jpg ${KERNELID}:/home/sigmaos/7.jpg
    # docker cp ~/8.jpg ${KERNELID}:/home/sigmaos/8.jpg
  else
    echo "JOIN ${SIGMASTART} ${KERNELID}"
    ${TOKEN} 2>&1 > /dev/null
    ./start-kernel.sh --boot node --named ${SIGMASTART_PUBADDR} --pull ${TAG} --dbip ${MAIN_PRIVADDR}:4406 --mongoip ${MAIN_PRIVADDR}:4407 ${OVERLAYS} --provider ${PROVIDER} ${KERNELID} 2>&1 | tee /tmp/join.out
    docker cp ~/1.jpg ${KERNELID}:/home/sigmaos/1.jpg
    docker cp ~/6.jpg ${KERNELID}:/home/sigmaos/6.jpg
    docker cp ~/7.jpg ${KERNELID}:/home/sigmaos/7.jpg
    # docker cp ~/8.jpg ${KERNELID}:/home/sigmaos/8.jpg
  fi
ENDSSH
  if [ "${vm}" = "${MAIN}" ]; then
    TOKEN=$(ssh -i $DIR/keys/cloudlab-sigmaos $LOGIN@$vm docker swarm join-token worker | grep docker)
  fi   
done
