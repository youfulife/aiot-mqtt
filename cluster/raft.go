package cluster

import (
    "io"
    "net"
    "os"
    "path/filepath"
    "time"

    "github.com/hashicorp/raft"

    raftboltdb "github.com/hashicorp/raft-boltdb"
)

type Fsm struct {
}

func NewFsm() *Fsm {
    fsm := &Fsm{}
    return fsm
}

func (f *Fsm) Apply(l *raft.Log) interface{} {
    return nil
}

func (f *Fsm) Snapshot() (raft.FSMSnapshot, error) {
    return nil, nil
}

func (f *Fsm) Restore(io.ReadCloser) error {
    return nil
}

func NewRaft(raftAddr, raftId, raftDir string, bootstrap bool) (*raft.Raft, raft.FSM, error) {
    config := raft.DefaultConfig()
    config.LocalID = raft.ServerID(raftId)

    addr, err := net.ResolveTCPAddr("tcp", raftAddr)
    if err != nil {
        return nil, nil, err
    }
    transport, err := raft.NewTCPTransport(raftAddr, addr, 2, 5*time.Second, os.Stderr)
    if err != nil {
        return nil, nil, err
    }

    snapshots, err := raft.NewFileSnapshotStore(raftDir, 2, os.Stderr)
    if err != nil {
        return nil, nil, err
    }

    logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
    if err != nil {
        return nil, nil, err
    }

    stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.db"))
    if err != nil {
        return nil, nil, err
    }

    fsm := NewFsm()
    r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
    if err != nil {
        return nil, nil, err
    }

    if bootstrap {
        configuration := raft.Configuration{
            Servers: []raft.Server{
                {
                    ID:      config.LocalID,
                    Address: transport.LocalAddr(),
                },
            },
        }
        r.BootstrapCluster(configuration)
    }

    return r, fsm, nil
}
