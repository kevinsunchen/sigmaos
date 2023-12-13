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

./start-sigmaos.sh --vpc vpc-08531d701dd868b70 --branch etcd-sigmasrv-newprocclnt-prvdr --pull kschen-sigmaos

cd $CLOUDLAB_DIR
./start-sigmaos-subprvdr.sh --vpc vpc-08531d701dd868b70 --branch etcd-sigmasrv-newprocclnt-prvdr --pull kschen-sigmaos

cd $AWS_DIR
ssh -i key-$VPC.pem ubuntu@$LEADER_PUBLIC_IP <<ENDSSH
  cd sigmaos
  export SIGMADEBUG="BENCH;IMGD;"
  go clean -testcache
  go test -v sigmaos/benchmarks --tag kschen-sigmaos --run TestImgResizeMultiProvider --n_imgresizemp_each 10 --imgresize_nround 25 --imgresizemp_providers_to_paths "name/s3/~local/kschen-9ps3/img/1.jpg:cloudlab;name/s3/~local/kschen-9ps3/img/6.jpg:aws;name/s3/~local/kschen-9ps3/img/7.jpg:aws" --imgresize_mcpu 500 --imgresize_mem 0 --imgresizemp_init_provider aws
ENDSSH

cd $CLOUDLAB_DIR
./stop-sigmaos.sh

cd $AWS_DIR
./stop-sigmaos.sh --vpc vpc-08531d701dd868b70


# cd $MP_DIR

# ./stop-sigmaos-mp.sh
# export SIGMADEBUG="TEST;BENCH;IMGD;GROUPMGR;"
# ./start-sigmaos-mp.sh

# # cd $MP_DIR
# # ./stop-sigmaos-mp.sh --vpc $VPC
