package nano

type (
	options struct {
		pipeline Pipeline
	}

	Option func(*options)
)

func WithPipeline(pipeline Pipeline) Option {
	return func(opt *options) {
		opt.pipeline = pipeline
	}
}
