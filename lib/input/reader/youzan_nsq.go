package reader

import (
	"io/ioutil"
	llog "log"
	"sync"
	"time"

	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/message"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/types"

	"github.com/youzan/go-nsq"
)

type YouZanNSQConfig struct {
	LookupAddresses []string `json:"lookupd_http_addresses" yaml:"lookupd_http_addresses"`
	Topic           string   `json:"topic" yaml:"topic"`
	Channel         string   `json:"channel" yaml:"channel"`
	UserAgent       string   `json:"user_agent" yaml:"user_agent"`
	MaxInFlight     int      `json:"max_in_flight" yaml:"max_in_flight"`
	EnableOrdered   bool     `json:"enable_ordered" yaml:"enable_ordered"`
}

func NewYouZanNSQConfig() YouZanNSQConfig {
	return YouZanNSQConfig{
		LookupAddresses: []string{"localhost:4161"},
		Topic:           "benthos_messages",
		Channel:         "benthos_stream",
		UserAgent:       "benthos_consumer",
		MaxInFlight:     100,
		EnableOrdered:   false,
	}
}

type YouZanNSQ struct {
	cMut     sync.Mutex
	consumer *nsq.Consumer

	lookupAddresses []string

	logger log.Modular
	conf   YouZanNSQConfig
	stats  metrics.Type

	internalMessages chan *nsq.Message
	interruptChan    chan struct{}
	unAckMessages    []*nsq.Message
}

func NewYouZanNSQ(conf YouZanNSQConfig, log log.Modular, stats metrics.Type) (Type, error) {
	n := YouZanNSQ{
		lookupAddresses: conf.LookupAddresses,
		conf:            conf,
		logger:          log,
		stats:           stats,

		internalMessages: make(chan *nsq.Message),
		interruptChan:    make(chan struct{}),
	}

	return &n, nil
}

func (n *YouZanNSQ) Connect() error {
	n.cMut.Lock()
	defer n.cMut.Unlock()

	if n.consumer != nil {
		return nil
	}

	consumerConfig := nsq.NewConfig()
	consumerConfig.UserAgent = n.conf.UserAgent
	consumerConfig.MaxInFlight = n.conf.MaxInFlight
	consumerConfig.EnableOrdered = n.conf.EnableOrdered

	consumer, err := nsq.NewConsumer(n.conf.Topic, n.conf.Channel, consumerConfig)
	if err != nil {
		return err
	}
	consumer.SetLogger(llog.New(ioutil.Discard, "", llog.Flags()), nsq.LogLevelError)
	consumer.AddHandler(n)

	if err = consumer.ConnectToNSQLookupds(n.lookupAddresses); err != nil {
		consumer.Stop()
		return err
	}

	n.consumer = consumer
	n.logger.Infof("Receiving YouZanNSQ messages from lookupAddresses: %s\n", n.lookupAddresses)
	return nil
}

func (n *YouZanNSQ) disconnect() error {
	n.cMut.Lock()
	defer n.cMut.Unlock()

	if n.consumer != nil {
		n.consumer.Stop()
		<-n.consumer.StopChan
		n.consumer = nil
	}
	return nil
}

func (n *YouZanNSQ) HandleMessage(message *nsq.Message) error {
	message.DisableAutoResponse()
	select {
	case n.internalMessages <- message:
	case <-n.interruptChan:
		message.Requeue(-1)
	}
	return nil
}

func (n *YouZanNSQ) Acknowledge(err error) error {
	for _, m := range n.unAckMessages {
		if err != nil {
			m.Requeue(-1)
		} else {
			m.Finish()
		}
	}

	n.unAckMessages = nil
	return nil
}

func (n *YouZanNSQ) Read() (types.Message, error) {
	var msg *nsq.Message
	select {
	case msg = <-n.internalMessages:
		n.unAckMessages = append(n.unAckMessages, msg)
	case <-n.interruptChan:
		for _, m := range n.unAckMessages {
			m.Requeue(-1)
		}
		n.unAckMessages = nil
		_ = n.disconnect()
		return nil, types.ErrTypeClosed
	}
	return message.New([][]byte{msg.Body}), nil
}

func (n *YouZanNSQ) CloseAsync() {
	close(n.interruptChan)
}

func (n *YouZanNSQ) WaitForClose(timeout time.Duration) error {
	if n.consumer != nil {
		<-n.consumer.StopChan
	}
	return nil
}
