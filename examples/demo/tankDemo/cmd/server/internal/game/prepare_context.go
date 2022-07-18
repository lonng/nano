package game

type prepareContext struct {
	readyStatus map[int64]bool //是否已经ready完毕
}

func newPrepareContext() *prepareContext {
	return &prepareContext{
		readyStatus: map[int64]bool{},
	}
}

func (p *prepareContext) isReady(uid int64) bool {
	return p.readyStatus[uid]
}

func (p *prepareContext) ready(uid int64) {
	p.readyStatus[uid] = true
}

func (p *prepareContext) reset() {
	p.readyStatus = map[int64]bool{}
}
