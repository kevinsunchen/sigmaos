for h in $(cat servers.txt | cut -d " " -f 2); do
	echo "=========== Setting up instance for $h";
	./setup-instance.sh $h;
done

./update-repo.sh --branch etcd-sigmasrv-newprocclnt-prvdr