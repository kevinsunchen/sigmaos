package session

import (
	"encoding/json"

	db "ulambda/debug"
	np "ulambda/ninep"
	"ulambda/threadmgr"
)

func (st *SessionTable) Snapshot() []byte {
	sessions := make(map[np.Tsession][]byte)
	for sid, sess := range st.sessions {
		sessions[sid] = sess.Snapshot()
	}
	b, err := json.Marshal(sessions)
	if err != nil {
		db.DFatalf("Error snapshot encoding session table: %v", err)
	}
	return b
}

func RestoreTable(oldSt *SessionTable, mkps np.MkProtServer, rps np.RestoreProtServer, fssrv np.FsServer, tm *threadmgr.ThreadMgrTable, b []byte) *SessionTable {
	sessions := make(map[np.Tsession][]byte)
	err := json.Unmarshal(b, &sessions)
	if err != nil {
		db.DFatalf("error unmarshal session table in restore: %v", err)
	}
	st := MakeSessionTable(mkps, fssrv, tm)
	for sid, b := range sessions {
		st.sessions[sid] = RestoreSession(sid, fssrv, rps, tm, b)
		// Set the replies channel if this sesison already exists at this replica
		if oldSess, ok := oldSt.Lookup(sid); ok {
			st.sessions[sid].SetConn(oldSess.GetConn())
		}
	}
	return st
}

type SessionSnapshot struct {
	ProtsrvSnap []byte
	closed      bool
}

func MakeSessionSnapshot() *SessionSnapshot {
	return &SessionSnapshot{}
}

func (sess *Session) Snapshot() []byte {
	ss := MakeSessionSnapshot()
	ss.ProtsrvSnap = sess.protsrv.Snapshot()
	ss.closed = sess.closed
	b, err := json.Marshal(ss)
	if err != nil {
		db.DFatalf("Error snapshot encoding session: %v", err)
	}
	return b
}

func RestoreSession(sid np.Tsession, fssrv np.FsServer, rps np.RestoreProtServer, tmt *threadmgr.ThreadMgrTable, b []byte) *Session {
	ss := MakeSessionSnapshot()
	err := json.Unmarshal(b, ss)
	if err != nil {
		db.DFatalf("error unmarshal session in restore: %v", err)
	}
	fos := rps(fssrv, ss.ProtsrvSnap)
	// TODO: add session manager
	sess := makeSession(fos, sid, tmt.AddThread())
	sess.closed = ss.closed
	return sess
}
