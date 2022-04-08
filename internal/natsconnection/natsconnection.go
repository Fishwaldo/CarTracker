package natsconnection

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/nats-io/nats.go"
	"github.com/sasha-s/go-deadlock"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("nats.host", "nats.example.com")
	viper.SetDefault("nats.port", 4222)
	viper.SetDefault("nats.credfile", "config/nats.creds")
}

type NatsConnS struct {
	conn *nats.Conn
	enc *nats.EncodedConn
	js nats.JetStreamContext
	logger logr.Logger
	inCMDSubject string
	inCmdSubscription *nats.Subscription
	outCMDPrefix string
	mx deadlock.Mutex
}

var Nats NatsConnS

func (nc *NatsConnS) Start(log logr.Logger) {
	nc.logger = log

	url := fmt.Sprintf("%s:%d", viper.GetString("nats.host"), viper.GetInt("nats.port"))
	var err error
	var options []nats.Option

	if _, err := os.Stat(viper.GetString("nats.credfile")); os.IsNotExist(err) {
		nc.logger.Error(err, "Credential File Does not Exist")
	} else {
		options = append(options, nats.UserCredentials(viper.GetString("nats.credfile")))
	}
	options = append(options, nats.RetryOnFailedConnect(true))
	options = append(options, nats.Name(viper.GetString("name")))
	//options = append(options, nats.NoEcho())
	options = append(options, nats.DisconnectErrHandler(nc.serverDisconnect))
	options = append(options, nats.ReconnectHandler(nc.serverReconnected))
	options = append(options, nats.ReconnectBufSize(64*1024*1024))

	if nc.conn, err = nats.Connect(url, options...); err != nil {
		nc.logger.Error(err, "Can't Connect to NATS Server")
	}
	nc.logger.Info("Connected to NATS Server", "name", nc.conn.ConnectedServerName(), "cluster", nc.conn.ConnectedClusterName())
	if nc.enc, err = nats.NewEncodedConn(nc.conn, "json"); err != nil {
		nc.logger.Error(err, "Can't Create Encoded Connection")
	}

	if nc.js, err = nc.conn.JetStream(); err != nil {
		nc.logger.Error(err, "Can't Create JetStream Context")
	}

	nc.inCMDSubject = fmt.Sprintf("cmd.car.%s", viper.GetString("name"))
	nc.outCMDPrefix = fmt.Sprintf("report.car.%s", viper.GetString("name"))
	if nc.inCmdSubscription, err = nc.enc.Subscribe(nc.inCMDSubject, nc.gotMessage); err != nil {
		nc.logger.Error(err, "Can't Subscribe to subject", "subject", nc.inCMDSubject)
	}
	if err = nc.enc.Publish(nc.inCMDSubject, string("test hello")); err != nil {
		nc.logger.Error(err, "Can't Publish to subject", "subject", nc.inCMDSubject)
	}

	nc.conn.Flush()
}

func (nc *NatsConnS) gotMessage(m *nats.Msg) {
	nc.logger.Info("Got Message from Subject", "subject", m.Subject, "data", m.Data)
}

func (nc *NatsConnS) serverDisconnect(c *nats.Conn, err error) {
	nc.logger.Error(err, "Nats Server Disconnected")
}
func (nc *NatsConnS) serverReconnected(c *nats.Conn) {
	nc.logger.Info("Nats Server Reconnected", "name", nc.conn.ConnectedServerName(), "cluster", nc.conn.ConnectedClusterName())
}

func (nc *NatsConnS) SendStats(domain string, ps interface{}) {
	nc.mx.Lock()
	defer nc.mx.Unlock()
	msg := nats.NewMsg(fmt.Sprintf("%s.%s", nc.outCMDPrefix, domain))
	msg.Header.Add("X-Msg-Time", time.Now().Format(time.RFC3339))
	msg.Data, _ = json.Marshal(ps)
	if _, err := nc.js.PublishMsg(msg); err != nil {
		nc.logger.Error(err, "Can't Publish Messages", "domain", domain)
	}
}

