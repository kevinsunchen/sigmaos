package fslib

import (
	"fmt"
	"strings"
	// "log"

	// db "sigmaos/debug"
	np "sigmaos/sigmap"
)

func MakeTarget(srvaddrs []string) []byte {
	targets := []string{}
	for _, addr := range srvaddrs {
		targets = append(targets, addr+":pubkey")
	}
	return []byte(strings.Join(targets, "\n"))
}

func MakeTargetTree(srvaddr string, tree np.Path) []byte {
	target := []string{srvaddr, "pubkey", tree.String()}
	return []byte(strings.Join(target, ":"))
}

func (fsl *FsLib) PostService(srvaddr, srvname string) error {
	err := fsl.Symlink(MakeTarget([]string{srvaddr}), srvname, 0777|np.DMTMP)
	return err
}

func (fsl *FsLib) PostServiceUnion(srvaddr, srvpath, server string) error {
	p := srvpath + "/" + server
	dir, err := fsl.IsDir(srvpath)
	if err != nil {
		return err
	}
	if !dir {
		return fmt.Errorf("Not a directory")
	}
	err = fsl.Symlink(MakeTarget([]string{srvaddr}), p, 0777|np.DMTMP)
	return err
}

func (fsl *FsLib) Post(srvaddr, path string) error {
	if np.EndSlash(path) {
		return fsl.PostServiceUnion(srvaddr, path, srvaddr)
	} else {
		return fsl.PostService(srvaddr, path)
	}
}
