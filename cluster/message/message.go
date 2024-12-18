package message

import "encoding/json"

//go:generate msgp
type Message struct {
    Type            byte   `json:"type" msg:"type"`
    NodeID          string `json:"node-id" msg:"node-id"`
    ClientID        string `json:"client-id" msg:"client-id"`
    ProtocolVersion byte   `json:"protocol-version" msg:"protocol-version"`
    Payload         []byte `json:"payload" msg:"payload"`
}

func (m *Message) JsonBytes() []byte {
    data, err := json.Marshal(m)
    if err != nil {
        return []byte("")
    }

    return data
}

func (m *Message) JsonLoad(data []byte) error {
    if err := json.Unmarshal(data, m); err != nil {
        return err
    }

    return nil
}

func (m *Message) MsgpackBytes() []byte {
    data, err := m.MarshalMsg(nil)
    if err != nil {
        return []byte("")
    }

    return data
}

func (m *Message) MsgpackLoad(data []byte) error {
    if _, err := m.UnmarshalMsg(data); err != nil {
        return err
    }

    return nil
}
