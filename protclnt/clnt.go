package protclnt

import (
	"sigmaos/path"
	"sigmaos/rand"
	"sigmaos/serr"
	"sigmaos/sessclnt"
	"sigmaos/sessp"
	sp "sigmaos/sigmap"
)

// Each proc has a unique client ID.
var clid sessp.Tclient = 0

func init() {
	if clid == 0 {
		clid = sessp.Tclient(rand.Uint64())
	}
}

type Clnt struct {
	id sessp.Tclient
	sm *sessclnt.Mgr
}

func NewClnt(clntnet string) *Clnt {
	clnt := &Clnt{}
	clnt.id = clid
	clnt.sm = sessclnt.NewMgr(clnt.id, clntnet)
	return clnt
}

func (clnt *Clnt) CallServer(addrs sp.Taddrs, args sessp.Tmsg, data []byte) (*sessp.FcallMsg, *serr.Err) {
	reply, err := clnt.sm.RPC(addrs, args, data)
	if err != nil {
		return nil, err
	}
	rmsg, ok := reply.Msg.(*sp.Rerror)
	if ok {
		return nil, sp.NewErr(rmsg)
	}
	return reply, nil
}

func (clnt *Clnt) Attach(addrs sp.Taddrs, uname sp.Tuname, cid sp.TclntId, fid sp.Tfid, path path.Path) (*sp.Rattach, *serr.Err) {
	args := sp.NewTattach(fid, sp.NoFid, uname, cid, path)
	reply, err := clnt.CallServer(addrs, args, nil)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rattach)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "clnt")
	}
	return msg, nil
}

func (clnt *Clnt) DetachAll(cid sp.TclntId) *serr.Err {
	scs := clnt.sm.SessClnts()
	for _, sc := range scs {
		sc.Detach(cid)
	}
	return nil
}

func (clnt *Clnt) NewProtClnt(addrs sp.Taddrs) *ProtClnt {
	protclnt := &ProtClnt{addrs, clnt}
	return protclnt
}

type ProtClnt struct {
	addrs sp.Taddrs
	clnt  *Clnt
}

func (pclnt *ProtClnt) Servers() sp.Taddrs {
	return pclnt.addrs
}

func (pclnt *ProtClnt) Disconnect() *serr.Err {
	return pclnt.clnt.sm.Disconnect(pclnt.addrs)
}

func (pclnt *ProtClnt) Call(args sessp.Tmsg) (*sessp.FcallMsg, *serr.Err) {
	return pclnt.clnt.CallServer(pclnt.addrs, args, nil)
}

func (pclnt *ProtClnt) CallData(args sessp.Tmsg, data []byte) (*sessp.FcallMsg, *serr.Err) {
	return pclnt.clnt.CallServer(pclnt.addrs, args, data)
}

func (pclnt *ProtClnt) Walk(fid sp.Tfid, nfid sp.Tfid, path path.Path) (*sp.Rwalk, *serr.Err) {
	args := sp.NewTwalk(fid, nfid, path)
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rwalk)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rwalk")
	}
	return msg, nil
}

func (pclnt *ProtClnt) Create(fid sp.Tfid, name string, perm sp.Tperm, mode sp.Tmode, lid sp.TleaseId, f sp.Tfence) (*sp.Rcreate, *serr.Err) {
	args := sp.NewTcreate(fid, name, perm, mode, lid, f)
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rcreate)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rcreate")
	}
	return msg, nil
}

func (pclnt *ProtClnt) Remove(fid sp.Tfid) *serr.Err {
	args := sp.NewTremove(fid, sp.NullFence())
	reply, err := pclnt.Call(args)
	if err != nil {
		return err
	}
	_, ok := reply.Msg.(*sp.Rremove)
	if !ok {
		return serr.NewErr(serr.TErrBadFcall, "Rremove")
	}
	return nil
}

func (pclnt *ProtClnt) RemoveF(fid sp.Tfid, f *sp.Tfence) *serr.Err {
	args := sp.NewTremove(fid, f)
	reply, err := pclnt.Call(args)
	if err != nil {
		return err
	}
	_, ok := reply.Msg.(*sp.Rremove)
	if !ok {
		return serr.NewErr(serr.TErrBadFcall, "Rremove")
	}
	return nil
}

func (pclnt *ProtClnt) RemoveFile(fid sp.Tfid, wnames path.Path, resolve bool, f *sp.Tfence) *serr.Err {
	args := sp.NewTremovefile(fid, wnames, resolve, f)
	reply, err := pclnt.Call(args)
	if err != nil {
		return err
	}
	_, ok := reply.Msg.(*sp.Rremove)
	if !ok {
		return serr.NewErr(serr.TErrBadFcall, "Rremovefile")
	}
	return nil
}

func (pclnt *ProtClnt) Clunk(fid sp.Tfid) *serr.Err {
	args := sp.NewTclunk(fid)
	reply, err := pclnt.Call(args)
	if err != nil {
		return err
	}
	_, ok := reply.Msg.(*sp.Rclunk)
	if !ok {
		return serr.NewErr(serr.TErrBadFcall, "Rclunk")
	}
	return nil
}

func (pclnt *ProtClnt) Open(fid sp.Tfid, mode sp.Tmode) (*sp.Ropen, *serr.Err) {
	args := sp.NewTopen(fid, mode)
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Ropen)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Ropen")
	}
	return msg, nil
}

func (pclnt *ProtClnt) Watch(fid sp.Tfid) *serr.Err {
	args := sp.NewTwatch(fid)
	reply, err := pclnt.Call(args)
	if err != nil {
		return err
	}
	_, ok := reply.Msg.(*sp.Ropen)
	if !ok {
		return serr.NewErr(serr.TErrBadFcall, "Rwatch")
	}
	return nil
}

func (pclnt *ProtClnt) ReadF(fid sp.Tfid, offset sp.Toffset, cnt sp.Tsize, f *sp.Tfence) ([]byte, *serr.Err) {
	args := sp.NewReadF(fid, offset, cnt, f)
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	_, ok := reply.Msg.(*sp.Rread)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rread")
	}
	return reply.Data, nil
}

func (pclnt *ProtClnt) WriteF(fid sp.Tfid, offset sp.Toffset, f *sp.Tfence, data []byte) (*sp.Rwrite, *serr.Err) {
	args := sp.NewTwriteF(fid, offset, f)
	reply, err := pclnt.CallData(args, data)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rwrite)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rwrite")
	}
	return msg, nil
}

func (pclnt *ProtClnt) WriteRead(fid sp.Tfid, data []byte) ([]byte, *serr.Err) {
	args := sp.NewTwriteread(fid)
	reply, err := pclnt.CallData(args, data)
	if err != nil {
		return nil, err
	}
	_, ok := reply.Msg.(*sp.Rread)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rwriteread")
	}
	return reply.Data, nil
}

func (pclnt *ProtClnt) Stat(fid sp.Tfid) (*sp.Rstat, *serr.Err) {
	args := sp.NewTstat(fid)
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rstat)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rstat")
	}
	return msg, nil
}

func (pclnt *ProtClnt) Wstat(fid sp.Tfid, st *sp.Stat) (*sp.Rwstat, *serr.Err) {
	args := sp.NewTwstat(fid, st, sp.NullFence())
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rwstat)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rwstat")
	}
	return msg, nil
}

func (pclnt *ProtClnt) WstatF(fid sp.Tfid, st *sp.Stat, f *sp.Tfence) (*sp.Rwstat, *serr.Err) {
	args := sp.NewTwstat(fid, st, f)
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rwstat)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rwstat")
	}
	return msg, nil
}

func (pclnt *ProtClnt) Renameat(oldfid sp.Tfid, oldname string, newfid sp.Tfid, newname string, f *sp.Tfence) (*sp.Rrenameat, *serr.Err) {
	args := sp.NewTrenameat(oldfid, oldname, newfid, newname, f)
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rrenameat)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rrenameat")
	}
	return msg, nil
}

func (pclnt *ProtClnt) GetFile(fid sp.Tfid, path path.Path, mode sp.Tmode, offset sp.Toffset, cnt sp.Tsize, resolve bool, f *sp.Tfence) ([]byte, *serr.Err) {
	args := sp.NewTgetfile(fid, mode, offset, cnt, path, resolve, f)
	reply, err := pclnt.Call(args)
	if err != nil {
		return nil, err
	}
	_, ok := reply.Msg.(*sp.Rread)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rgetfile")
	}
	return reply.Data, nil
}

func (pclnt *ProtClnt) PutFile(fid sp.Tfid, path path.Path, mode sp.Tmode, perm sp.Tperm, offset sp.Toffset, resolve bool, f *sp.Tfence, data []byte, lid sp.TleaseId) (*sp.Rwrite, *serr.Err) {
	args := sp.NewTputfile(fid, mode, perm, offset, path, resolve, lid, f)
	reply, err := pclnt.CallData(args, data)
	if err != nil {
		return nil, err
	}
	msg, ok := reply.Msg.(*sp.Rwrite)
	if !ok {
		return nil, serr.NewErr(serr.TErrBadFcall, "Rwrite")
	}
	return msg, nil
}

func (pclnt *ProtClnt) Detach(cid sp.TclntId) *serr.Err {
	args := sp.NewTdetach(cid)
	_, err := pclnt.Call(args)
	if err != nil {
		return err
	}
	return nil
}
