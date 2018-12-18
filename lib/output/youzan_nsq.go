package output

import (
	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/output/writer"
	"github.com/Jeffail/benthos/lib/types"
)

func init() {
	Constructors[TypeYouZanNSQ] = TypeSpec{
		constructor: NewYouZanNSQ,
		description: `
Publish to an YouZan NSQ topic. The ` + "`topic`" + ` field can be dynamically set
using function interpolations described
[here](../config_interpolation.md#functions). When sending batched messages
these interpolations are performed per message part.`,
	}
}

func NewYouZanNSQ(conf Config, mgr types.Manager, log log.Modular, stats metrics.Type) (Type, error) {
	w, err := writer.NewYouZanNSQ(conf.YouZanNSQ, log, stats)
	if err != nil {
		return nil, err
	}
	return NewWriter("youzan_nsq", w, log, stats)
}
