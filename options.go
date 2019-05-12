package nano

import (
	"net/http"
	"time"

	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/message"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/serialize"
	"google.golang.org/grpc"
)

type (
	options struct {
		pipeline      pipeline.Pipeline
		advertiseAddr string
		memberAddr    string
		isMaster      bool
		grpcOptions   []grpc.DialOption
		components    *component.Components
	}

	Option func(*options)
)

func WithPipeline(pipeline pipeline.Pipeline) Option {
	return func(opt *options) {
		opt.pipeline = pipeline
	}
}

// WithAdvertiseAddr sets the advertise address option, it will be the listen address in
// master node and an advertise address which cluster member to connect
func WithAdvertiseAddr(addr string) Option {
	return func(opt *options) {
		opt.advertiseAddr = addr
	}
}

// WithMemberAddr sets the listen address which is used to establish connection between
// cluster members. Will select an available port automatically if no member address
// setting and panic if no available port
func WithMemberAddr(addr string) Option {
	return func(opt *options) {
		opt.memberAddr = addr
	}
}

// WithMaster sets the option to indicate whether the current node is master node
func WithMaster() Option {
	return func(opt *options) {
		opt.isMaster = true
	}
}

// WithGrpcOptions sets the grpc dial options
func WithGrpcOptions(opts ...grpc.DialOption) Option {
	return func(opt *options) {
		opt.grpcOptions = opts
	}
}

// WithComponents sets the components
func WithComponents(components *component.Components) Option {
	return func(opt *options) {
		opt.components = components
	}
}

// WithHeartbeatInterval sets Heartbeat time interval
func WithHeartbeatInterval(d time.Duration) Option {
	return func(_ *options) {
		env.Heartbeat = d
	}
}

// WithCheckOriginFunc sets the function that check `Origin` in http headers
func WithCheckOriginFunc(fn func(*http.Request) bool) Option {
	return func(opt *options) {
		env.CheckOrigin = fn
	}
}

// WithDebugMode let 'nano' to run under Debug mode.
func WithDebugMode() Option {
	return func(_ *options) {
		env.Debug = true
	}
}

// SetDictionary sets routes map
func WithDictionary(dict map[string]uint16) Option {
	return func(_ *options) {
		message.SetDictionary(dict)
	}
}

func WithWSPath(path string) Option {
	return func(_ *options) {
		env.WSPath = path
	}
}

// SetTimerPrecision set the ticker precision, and time precision can not less
// than a Millisecond, and can not change after application running. The default
// precision is time.Second
func WithTimerPrecision(precision time.Duration) Option {
	if precision < time.Millisecond {
		panic("time precision can not less than a Millisecond")
	}
	return func(_ *options) {
		env.TimerPrecision = precision
	}
}

// WithSerializer customizes application serializer, which automatically Marshal
// and UnMarshal handler payload
func WithSerializer(serializer serialize.Serializer) Option {
	return func(opt *options) {
		env.Serializer = serializer
	}
}
