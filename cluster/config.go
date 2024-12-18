package cluster

import (
    "fmt"

    "github.com/davecgh/go-spew/spew"
    "github.com/spf13/viper"
)

type ClusterConf struct {
    NodeName             string            `mapstructure:"node-name" yaml:"node-name" json:"node-name"`
    BindAddr             string            `mapstructure:"bind-addr" yaml:"bind-addr" json:"bind-addr"`
    BindPort             int               `mapstructure:"bind-port" yaml:"bind-port" json:"bind-port"`
    Members              []string          `mapstructure:"members" yaml:"members" json:"members"`
    QueueDepth           int               `mapstructure:"queue-depth" yaml:"queue-depth" json:"queue-depth"`
    Tags                 map[string]string `mapstructure:"tags" yaml:"tags" json:"tags"`
    InboundPoolSize      int               `mapstructure:"inbound-pool-size" yaml:"inbound-pool-size" json:"inbound-pool-size"`
    OutboundPoolSize     int               `mapstructure:"outbound-pool-size" yaml:"outbound-pool-size" json:"outbound-pool-size"`
    InoutPoolNonblocking bool              `mapstructure:"inout-pool-nonblocking" yaml:"inout-pool-nonblocking" json:"inout-pool-nonblocking"`
    NodesFileDir         string            `mapstructure:"nodes-file-dir" yaml:"nodes-file-dir" json:"nodes-file-dir"`
}

var Config ClusterConf

func ConfigLoad(path string) *ClusterConf {

    var c *viper.Viper
    c = viper.New()
    c.SetConfigFile(path)

    err := c.ReadInConfig()
    if err != nil {
        err = fmt.Errorf("配置文件读取失败: %v", err)
        panic(err)
    }

    err = c.Sub("cluster").Unmarshal(&Config)
    if err != nil {
        err = fmt.Errorf("配置文件序列化失败: %v", err)
        panic(err)
    }

    spew.Dump(Config)

    return &Config
}
