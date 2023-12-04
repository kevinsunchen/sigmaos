package sigmap

import (
	"net"

	"google.golang.org/protobuf/proto"

	"sigmaos/serr"
)

func NullMount() Tmount {
	return Tmount{}
}

func NewMount(b []byte) (Tmount, *serr.Err) {
	mnt := NullMount()
	if err := proto.Unmarshal(b, &mnt); err != nil {
		return mnt, serr.NewErrError(err)
	}
	return mnt, nil
}

func (mnt *Tmount) SetTree(tree string) {
	mnt.Root = tree
}

func (mnt *Tmount) SetAddr(addr Taddrs) {
	mnt.Addr = addr
}

func (mnt Tmount) Marshal() ([]byte, error) {
	return proto.Marshal(&mnt)
}

func (mnt Tmount) Address() *Taddr {
	return mnt.Addr[0]
}

func NewMountService(srvaddrs Taddrs) Tmount {
	return Tmount{Addr: srvaddrs}
}

func NewMountServer(addr string) Tmount {
	addrs := NewTaddrs([]string{addr})
	return NewMountService(addrs)
}

func NewMountServerMultAddr(addrSlice []string) Tmount {
	addrs := NewTaddrs(addrSlice)
	return NewMountService(addrs)
}

func (mnt Tmount) TargetHostPort() (string, string, error) {
	return net.SplitHostPort(mnt.Addr[0].Addr)
}
