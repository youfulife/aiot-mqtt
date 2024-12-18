package cluster

import (
    "bytes"
    "log/slog"

    "github.com/hashicorp/raft"
    "github.com/hashicorp/serf/serf"
    mqtt "github.com/mochi-mqtt/server/v2"
    "github.com/mochi-mqtt/server/v2/cluster/message"
    "github.com/mochi-mqtt/server/v2/packets"
    "github.com/panjf2000/ants/v2"
)

type Agent struct {
    Id       string
    config   *ClusterConf
    serf     *serf.Serf
    raft     *raft.Raft
    fsm      raft.FSM
    server   *mqtt.Server
    raftPool *ants.Pool
    OutPool  *ants.Pool
    inPool   *ants.Pool

    eventCh chan serf.Event
}

func NewAgent(conf *ClusterConf) *Agent {
    return &Agent{
        Id:     conf.NodeName,
        config: conf,
    }
}

func (a *Agent) BindServer(server *mqtt.Server) {
    _ = server.AddHook(new(MqttEventHook), a)
    a.server = server
}

func (a *Agent) Bootstrap() (err error) {

    //bootstrap := joinAddr == ""
    //a.raft, a.fsm, err = NewRaft(raftAddr, a.Id, raftDir, bootstrap)
    //if err != nil {
    //    return
    //}

    // create outbound goroutine pool and outbound msg
    if a.OutPool, err = ants.NewPool(a.config.OutboundPoolSize, ants.WithNonblocking(a.config.InoutPoolNonblocking)); err != nil {
        return err
    }

    // create inbound goroutine pool and process inbound msg
    if a.inPool, err = ants.NewPool(a.config.InboundPoolSize, ants.WithNonblocking(a.config.InoutPoolNonblocking)); err != nil {
        return err
    }

    a.eventCh = make(chan serf.Event, 64)
    go a.EventLoop()

    err = a.setupSerf()
    if err != nil {
        return
    }

    return
}

func (a *Agent) BroadcastInCluster(pk *packets.Packet) {
    _ = a.OutPool.Submit(func() {
        a.broadcastInCluster(pk)
    })
    return
}

// ApplyInCluster applies use raft
func (a *Agent) ApplyInCluster(msg *message.Message) {
    _ = a.OutPool.Submit(func() {
        // todo: apply to other nodes
    })
    return
}

func (a *Agent) broadcastInCluster(pk *packets.Packet) {
    switch pk.FixedHeader.Type {
    case packets.Publish:
        a.broadcastPublish(pk)
    default:
        return
    }
}

func (a *Agent) broadcastPublish(pk *packets.Packet) {

    msg := message.Message{
        Type:            packets.Publish,
        NodeID:          a.Id,
        ClientID:        pk.Origin,
        ProtocolVersion: pk.ProtocolVersion,
    }

    var buf bytes.Buffer
    //pk.Mods.AllowResponseInfo = true
    if err := pk.PublishEncode(&buf); err != nil {
        return
    }
    msg.Payload = buf.Bytes()

    // use gossip to broadcast
    a.broadcastMessage(msg.MsgpackBytes())
    return
}

func (a *Agent) broadcastMessage(message []byte) {
    err := a.serf.UserEvent(a.Id, message, false)
    if err != nil {
        a.server.Log.Error("broadcast message failed", err)
    }
}

func (a *Agent) setupSerf() (err error) {
    serfConf := serf.DefaultConfig()
    serfConf.Init()
    serfConf.NodeName = a.config.NodeName
    serfConf.MemberlistConfig.BindAddr = a.config.BindAddr
    serfConf.MemberlistConfig.BindPort = a.config.BindPort
    serfConf.EventCh = a.eventCh

    node, err := serf.Create(serfConf)
    if err != nil {
        return
    }

    if a.config.Members != nil {
        if len(a.config.Members) > 0 {
            if _, err = node.Join(a.config.Members, true); err != nil {
                return err
            }
        }
    }

    a.serf = node
    return
}

func (a *Agent) EventLoop() {
    for event := range a.eventCh {
        switch e := event.(type) {
        case serf.MemberEvent:
            a.handleMemberEvent(e)
        case serf.UserEvent:
            a.handleUserEvent(e)
        case *serf.Query:
            a.handleQueryEvent(e)
        default:
            a.server.Log.Error("Unknown event type: %s", event.EventType().String())
        }
    }
}

func (a *Agent) handleMemberEvent(e serf.MemberEvent) {
    slog.Info("handle member event", "event", e.String())
}

func (a *Agent) readFixedHeader(b []byte, fh *packets.FixedHeader) error {
    err := fh.Decode(b[0])
    if err != nil {
        return err
    }

    fh.Remaining, _, err = packets.DecodeLength(bytes.NewReader(b[1:]))
    if err != nil {
        return err
    }

    return nil
}

func (a *Agent) handleUserEvent(e serf.UserEvent) {
    a.server.Log.Debug("handle user event", e.String())

    var m message.Message
    if err := m.MsgpackLoad(e.Payload); err != nil {
        a.server.Log.Error("decode user event failed", err)
        return
    }

    switch m.Type {
    case packets.Publish:
        a.server.Log.Debug("handle publish message")

        pk := packets.Packet{
            FixedHeader:     packets.FixedHeader{Type: packets.Publish},
            ProtocolVersion: m.ProtocolVersion,
            Origin:          m.ClientID,
        }

        if err := a.readFixedHeader(m.Payload, &pk.FixedHeader); err != nil {
            return
        }

        offset := len(m.Payload) - pk.FixedHeader.Remaining // Unpack fixedheader.
        err := pk.PublishDecode(m.Payload[offset:])
        if err != nil {
            a.server.Log.Error("decode publish message failed", err)
            return
        }

        a.server.PublishToSubscribers(pk)
    default:
        a.server.Log.Warn("unhandled default case")
    }

}

func (a *Agent) handleQueryEvent(e *serf.Query) {
    a.server.Log.Debug("handle query event", e.String())
}
