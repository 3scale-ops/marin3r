package envoy

import "github.com/golang/protobuf/proto"

type Resource interface {
	proto.Message
}
