# usage() {
#   echo "Usage: $0 [--vpc VPC]" 1>&2
# }

VPC=vpc-08531d701dd868b70
# while [[ $# -gt 0 ]]; do
#   key="$1"
#   case $key in
#   --vpc)
#     shift
#     VPC=$1
#     shift
#     ;;
#   -help)
#     usage
#     exit 0
#     ;;
#   *)
#     echo "Error: unexpected argument '$1'"
#     usage
#     exit 1
#     ;;
#   esac
# done

ROOT_DIR=$(realpath $(dirname $0)/..)
AWS_DIR=$ROOT_DIR/aws
CLOUDLAB_DIR=$ROOT_DIR/cloudlab

cd $AWS_DIR

echo "Stopping AWS cluster....."
./stop-sigmaos.sh --vpc $VPC

cd $CLOUDLAB_DIR

echo "Stopping CloudLab cluster....."
./stop-sigmaos.sh

