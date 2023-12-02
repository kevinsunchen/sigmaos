#!/bin/bash

#
# Start proxy and mount root named at /mnt/9p. Optionally boot a
# rootrealm named.
#
# XXX keep IPaddr for now (maybe useful for overlays?)
#

usage() {
  echo "Usage: $0 --boot <IPaddr>"  1>&2
}

BOOT=""
while [ $# -ne 1 ]; do
    case "$1" in
        --boot)
            shift
            BOOT="--boot"
            ;;
        -help)
            usage
            exit 0
            ;;
        *)
            echo "unexpected argument $1"
            usage
            exit 1
            
    esac
done

if [[ "$BOOT" == "--boot" ]] ; then
    ./start-kernel.sh --boot all --named '127.0.0.1' sigma-named
fi

./bin/linux/proxyd $1 &

sleep 1

sudo mount -t 9p -o trans=tcp,aname=`whoami`,uname=`whoami`,port=1110 $1 /mnt/9p
