VPC="vpc-08531d701dd868b70"
BRANCH="etcd-sigmasrv-newprocclnt-prvdr"
TAG="kschen-sigmaos"

ROOT_DIR=$(realpath $(dirname $0)/../..)
AWS_DIR=$ROOT_DIR/aws
CLOUDLAB_DIR=$ROOT_DIR/cloudlab

cd $AWS_DIR
LEADER_IP_SIGMA=$(./leader-ip.sh --vpc $VPC)
LEADER_IP=$LEADER_IP_SIGMA

echo "Starting AWS cluster....."
./stop-sigmaos.sh --vpc $VPC
./start-sigmaos.sh --vpc $VPC --branch $BRANCH --pull $TAG
echo "AWS cluster started!"

cd $ROOT_DIR

echo "Starting CloudLab cluster....."
cd $CLOUDLAB_DIR
./stop-sigmaos.sh
./start-sigmaos-subprvdr.sh --vpc $VPC --branch $BRANCH --pull $TAG
echo "CloudLab cluster started!"
