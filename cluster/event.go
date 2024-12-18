package cluster

import (
    "bytes"
    "errors"

    mqtt "github.com/mochi-mqtt/server/v2"
    "github.com/mochi-mqtt/server/v2/packets"
)

// MqttEventHook 实现一个mqtt事件钩子，当发生连接、发布、订阅等事件时回调，同步给集群内其他节点。
type MqttEventHook struct {
    mqtt.HookBase
    agent *Agent
}

func (h *MqttEventHook) ID() string {
    return "agent-event"
}

func (h *MqttEventHook) Provides(b byte) bool {
    return bytes.Contains([]byte{
        mqtt.OnSessionEstablished,
        mqtt.OnSubscribed,
        mqtt.OnUnsubscribed,
        mqtt.OnPublished,
        mqtt.OnWillSent,
    }, []byte{b})
}

func (h *MqttEventHook) Init(config any) error {
    if _, ok := config.(*Agent); !ok && config != nil {
        return mqtt.ErrInvalidConfigType
    }

    if config == nil {
        return errors.New("MqttEventHook initialization failed")
    }

    h.agent = config.(*Agent)
    return nil
}

// OnSessionEstablished notifies other nodes to perform local subscription cleanup when their session is established.
func (h *MqttEventHook) OnSessionEstablished(cl *mqtt.Client, pk packets.Packet) {
    h.agent.BroadcastInCluster(&pk)
}

// OnWillSent is called when an LWT message has been issued from a disconnecting client.
func (h *MqttEventHook) OnWillSent(cl *mqtt.Client, pk packets.Packet) {
    h.agent.BroadcastInCluster(&pk)
}

// OnPublished is called when a client has published a message to subscribers.
func (h *MqttEventHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
    h.agent.BroadcastInCluster(&pk)
}

//
//// OnSubscribed is called when a client subscribes to one or more filters.
//func (h *MqttEventHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {
//    // 订阅消息需要通过 raft 同步到其他节点，这样一个publish的包来了之后，根据主题找到订阅的节点，然后发送给订阅的节点
//}
//
//// OnSubscribed is called when a client subscribes to one or more filters.
//func (h *MqttEventHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte, counts []int) {
//    if len(pk.Filters) == 0 {
//        return
//    }
//    for i, v := range pk.Filters {
//        if reasonCodes[i] <= packets.CodeGrantedQos2.Code && counts[i] == 1 { // first subscription
//            m := msg.Message{
//                Type:            packets.Subscribe,
//                ClientID:        cl.ID,
//                NodeID:          h.agent.GetLocalName(),
//                ProtocolVersion: cl.Properties.ProtocolVersion,
//                Payload:         []byte(v.Filter),
//            }
//            h.agent.SubmitRaftTask(&m)
//        }
//    }
//}
//
//func (h *MqttEventHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte, counts []int) {
//    if len(pk.Filters) == 0 {
//        return
//    }
//    for i, v := range pk.Filters {
//        if reasonCodes[i] == packets.CodeSuccess.Code && counts[i] == 0 { // first subscription
//            m := msg.Message{
//                Type:            packets.Unsubscribe,
//                ClientID:        cl.ID,
//                NodeID:          h.agent.GetLocalName(),
//                ProtocolVersion: cl.Properties.ProtocolVersion,
//                Payload:         []byte(v.Filter),
//            }
//            h.agent.SubmitRaftTask(&m)
//        }
//    }
//}
