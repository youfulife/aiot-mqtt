package main

import (
    "flag"
    "log"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    mqtt "github.com/mochi-mqtt/server/v2"
    "github.com/mochi-mqtt/server/v2/cluster"
    "github.com/mochi-mqtt/server/v2/config"
)

func setupMqttServer(configFile string) *mqtt.Server {
    configBytes, err := os.ReadFile(configFile)
    if err != nil {
        log.Fatal(err)
    }

    options, err := config.FromBytes(configBytes)
    if err != nil {
        log.Fatal(err)
    }

    server := mqtt.New(options)

    level := new(slog.LevelVar)
    server.Log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: level,
    }))
    level.Set(slog.LevelDebug)

    return server
}

func setupClusterAgent(configFile string) *cluster.Agent {

    conf := cluster.ConfigLoad(configFile)
    agent := cluster.NewAgent(conf)
    err := agent.Bootstrap()
    if err != nil {
        log.Fatal(err)
    }
    return agent
}

func main() {
    configFile := flag.String("config", "config.yaml", "path to mochi config yaml or json file")
    flag.Parse()

    sigs := make(chan os.Signal, 1)
    done := make(chan bool, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigs
        done <- true
    }()

    server := setupMqttServer(*configFile)
    agent := setupClusterAgent(*configFile)
    agent.BindServer(server)

    go func() {
        err := server.Serve()
        if err != nil {
            log.Fatal(err)
        }
    }()

    <-done
    server.Log.Warn("caught signal, stopping...")
    _ = server.Close()
    server.Log.Info("mochi mqtt shutdown complete")
}
