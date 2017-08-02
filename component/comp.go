package component

type Component interface {
	Init()
	AfterInit()
	BeforeShutdown()
	Shutdown()
}
