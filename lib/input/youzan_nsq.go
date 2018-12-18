package input

import (
	"github.com/Jeffail/benthos/lib/input/reader"
	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/types"
)

func init() {
	Constructors[TypeYouZanNSQ] = TypeSpec{
		constructor: NewYouZanNSQ,
		description: `
Subscribe to an YouZan NSQ instance topic and channel.`,
	}
}

func NewYouZanNSQ(conf Config, mgr types.Manager, log log.Modular, stats metrics.Type) (Type, error) {
	n, err := reader.NewYouZanNSQ(conf.YouZanNSQ, log, stats)
	if err != nil {
		return nil, err
	}
	return NewReader("you_zan_nsq", n, log, stats)
}
