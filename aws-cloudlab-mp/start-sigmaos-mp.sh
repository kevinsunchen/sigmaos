usage() {
  echo "Usage: $0 [--vpc VPC] [--branch BRANCH] [--pull TAG] [--n N_VM]" 1>&2
}

VPC=vpc-08531d701dd868b70
BRANCH=etcd-sigmasrv-newprocclnt-prvdr
TAG=kschen-sigmaos
N_VM=""
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

cd $AWS_DIR

echo "Starting AWS cluster....."
./start-sigmaos.sh --vpc $VPC --branch $BRANCH --pull $TAG
echo "AWS cluster started!"

cd $ROOT_DIR
cd $CLOUDLAB_DIR

echo "Starting CloudLab cluster....."
./start-sigmaos-subprvdr.sh --vpc $VPC --branch $BRANCH --pull $TAG
echo "CloudLab cluster started!"
