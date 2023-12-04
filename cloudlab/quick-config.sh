n=0
for h in $(cat servers.txt | cut -d " " -f 2); do
	echo "=========== Upgrading linux for node $n: $h";
    export BLKDEV="/dev/emulab/node$n-bs";
    echo $BLKDEV
	./configure-kernel.sh $h >& /tmp/$h.out;
    ((n=n+1));
done