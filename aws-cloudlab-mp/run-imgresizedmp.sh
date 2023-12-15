usage() {
  echo "Usage: $0 [--vpc VPC] [--branch BRANCH] [--pull TAG] [--n N_VM]" 1>&2
}

VPC=vpc-08531d701dd868b70
BRANCH=etcd-sigmasrv-newprocclnt-prvdr
TAG=kschen-sigmaos
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
  --pull)
    shift
    TAG=$1
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

ROOT_DIR=$(realpath $(dirname $0)/..)
MP_DIR=$ROOT_DIR/aws-cloudlab-mp
AWS_DIR=$ROOT_DIR/aws
CLOUDLAB_DIR=$ROOT_DIR/cloudlab
OUT_DIR=$MP_DIR/outfile.txt

cd $AWS_DIR
LEADER_PUBLIC_IP=$(./leader-public-ip.sh --vpc $VPC)

# rm $OUT_DIR > /dev/null
echo "" > $OUT_DIR

echo "Starting aws"
echo "============ START CLUSTER: AWS ============" >> $OUT_DIR
./start-sigmaos.sh --vpc vpc-08531d701dd868b70 --branch etcd-sigmasrv-newprocclnt-prvdr --pull kschen-sigmaos >> $OUT_DIR

cd $CLOUDLAB_DIR
echo "Starting cloudlab"
echo "============ START CLUSTER: CLOUDLAB ============" >> $OUT_DIR
./start-sigmaos-subprvdr.sh --vpc vpc-08531d701dd868b70 --branch etcd-sigmasrv-newprocclnt-prvdr --pull kschen-sigmaos >> $OUT_DIR

cd $AWS_DIR
ssh -i key-$VPC.pem ubuntu@$LEADER_PUBLIC_IP <<ENDSSH
  cd sigmaos
  export SIGMADEBUG="BENCH;IMGD;"
  go clean -testcache
  go test -v sigmaos/benchmarks --etcdIP $LEADER_PUBLIC_IP --tag kschen-sigmaos --run TestImgResizeMultiProvider --n_imgresizemp_each 10 --imgresize_nround 25 --imgresizemp_providers_to_paths "name/s3/~local/kschen-9ps3/img/1.jpg:cloudlab;name/s3/~local/kschen-9ps3/img/6.jpg:aws;name/s3/~local/kschen-9ps3/img/7.jpg:aws" --imgresize_mcpu 500 --imgresize_mem 0 --imgresizemp_init_provider aws
ENDSSH

cd $CLOUDLAB_DIR
echo "Stopping cloudlab"
echo "============ STOP CLUSTER: CLOUDLAB ============" >> $OUT_DIR
./stop-sigmaos.sh >> $OUT_DIR

cd $AWS_DIR
echo "Stopping aws"
echo "============ STOP CLUSTER: AWS ============" >> $OUT_DIR
./stop-sigmaos.sh --vpc vpc-08531d701dd868b70 >> $OUT_DIR


# cd $MP_DIR

# ./stop-sigmaos-mp.sh
# export SIGMADEBUG="TEST;BENCH;IMGD;GROUPMGR;"
# ./start-sigmaos-mp.sh

# # cd $MP_DIR
# # ./stop-sigmaos-mp.sh --vpc $VPC
