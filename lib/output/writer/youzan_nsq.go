package writer

import (
	"fmt"
	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/message"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/types"
	"github.com/Jeffail/benthos/lib/util/text"
	"github.com/spaolacci/murmur3"
	"github.com/youzan/go-nsq"
	"io/ioutil"
	llog "log"
	"sync"
	"time"
)

type YouZanNSQConfig struct {
	LookupAddresses string `json:"lookupd_http_addresses" yaml:"lookupd_http_addresses"`
	Topic           string `json:"topic" yaml:"topic"`
	UserAgent       string `json:"user_agent" yaml:"user_agent"`
	EnableOrdered   bool   `json:"enable_ordered" yaml:"enable_ordered"`
}

// NewNSQConfig creates a new NSQConfig with default values.
func NewYouZanNSQConfig() YouZanNSQConfig {
	return YouZanNSQConfig{
		LookupAddresses: "localhost:4161",
		Topic:           "benthos_messages",
		UserAgent:       "benthos_producer",
		EnableOrdered:   false,
	}
}

type YouZanNSQ struct {
	log log.Modular

	topicStr *text.InterpolatedString

	connMut  sync.RWMutex
	producer *nsq.TopicProducerMgr

	config YouZanNSQConfig
}

func NewYouZanNSQ(conf YouZanNSQConfig, log log.Modular, stats metrics.Type) (*YouZanNSQ, error) {
	n := YouZanNSQ{
		log:      log,
		config:   conf,
		topicStr: text.NewInterpolatedString(conf.Topic),
	}
	return &n, nil
}

func (n *YouZanNSQ) Connect() error {
	n.connMut.Lock()
	defer n.connMut.Unlock()

	cfg := nsq.NewConfig()
	cfg.UserAgent = n.config.UserAgent
	cfg.Hasher = murmur3.New32()

	producer, err := nsq.NewTopicProducerMgr([]string{n.config.Topic}, cfg)
	if err != nil {
		return err
	}
	producer.SetLogger(llog.New(ioutil.Discard, "", llog.Flags()), nsq.LogLevelError)

	if err = producer.ConnectToNSQLookupd(n.config.LookupAddresses); err != nil {
		producer.Stop()
		return err
	}

	n.producer = producer
	n.log.Infof("Sending YouZan NSQ messages to address: %s\n", n.config.LookupAddresses)
	return nil
}

func (n *YouZanNSQ) Write(msg types.Message) error {
	n.connMut.RLock()
	prod := n.producer
	n.connMut.RUnlock()

	if prod == nil {
		return types.ErrNotConnected
	}

	return msg.Iter(func(i int, p types.Part) error {
		return prod.Publish(n.topicStr.Get(message.Lock(msg, i)), p.Get())
	})
}

func (n *YouZanNSQ) CloseAsync() {
	fmt.Printf("async start\n")
	n.connMut.Lock()
	defer n.connMut.Unlock()
	if n.producer != nil {
		n.producer.Stop()
		n.producer = nil
	}
	fmt.Printf("async end\n")
}

func (n *YouZanNSQ) WaitForClose(timeout time.Duration) error {
	fmt.Printf("wait\n")
	return nil
}
