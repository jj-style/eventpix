package dependencies

import "github.com/google/wire"

var Provider = wire.NewSet(NatsProvider)
