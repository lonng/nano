package nano

import "google.golang.org/grpc"

type (
	options struct {
		pipeline      Pipeline
		advertiseAddr string
		isMaster      bool
		grpcOptions   []grpc.DialOption
	}

	Option func(*options)
)

func WithPipeline(pipeline Pipeline) Option {
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

// WithMaster sets the option whether current node is master node
func WithMaster() Option {
	return func(opt *options) {
		opt.isMaster = true
	}
}

func WithGrpcOptions(opts ...grpc.DialOption) Option {
	return func(opt *options) {
		opt.grpcOptions = opts
	}
}
