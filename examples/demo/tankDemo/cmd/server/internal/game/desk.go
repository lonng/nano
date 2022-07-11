package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lonng/nano"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"github.com/lonng/nanoserver/db"
	"github.com/lonng/nanoserver/db/model"
	"github.com/lonng/nanoserver/internal/game/history"
	"github.com/lonng/nanoserver/internal/game/mahjong"
	"github.com/lonng/nanoserver/pkg/async"
	"github.com/lonng/nanoserver/pkg/constant"
	"github.com/lonng/nanoserver/pkg/errutil"
	"github.com/lonng/nanoserver/pkg/room"
	"github.com/lonng/nanoserver/protocol"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	ResultIllegal = 0
	ResultZiMo    = 1
	ResultHu      = 2
	ResultPao     = 3
	ResultPei     = 4
)

const (
	illegalTurn   = -1
	deskDissolved = -1 // 桌子解散标记
	illegalTile   = -1
)

type Desk struct {
	clubId    int64                 // 俱乐部ID
	roomNo    room.Number           // 房间号
	deskID    int64                 // desk表的pk
	opts      *protocol.DeskOptions // 房间选项
	state     constant.DeskStatus   // 状态
	round     uint32                // 第n局
	creator   int64                 // 创建玩家UID
	createdAt int64                 // 创建时间
	players   []*Player
	group     *nano.Group // 组播通道
	die       chan struct{}

	allTiles      mahjong.Mahjong //所有麻将
	bankerTurn    int             //庄家方位
	curTurn       int             //当前方位
	nextTileIndex int             //下一tile在整副麻将数组中的索引,非Tile's rank.
	isNewRound    bool            //是否是每局的第一次出牌
	isFirstRound  bool            //是否是本桌的第一局牌
	knownTiles    map[int]int     //已知麻将: index -> count

	// TODO(Waring): 里面只能统计分值变化, 不能用于统计暗杠和巴杠, 因为转雨会将分支变化直接转给胡牌玩家
	scoreChanges map[int64][]*scoreChangeInfo // uid -> score,
	snapshot     *history.History             //为本局准备快照
	roundStats   history.RoundStats           //单局分值变化统计
	matchStats   history.MatchStats           //场分值变化统计

	wonPlayers map[int64]bool //是否胡牌
	paoPlayer  int64          //炮手
	isMakerSet bool           // 当局第一次设置庄家

	dissolve *dissolveContext // 解散相关状态(改状态只可能只逻辑线程中更新, 不会并发)
	prepare  *prepareContext  // 准备相关状态
	dice     *dice            // 骰子

	lastTileId    int   //最后一张出牌
	lastChuPaiUid int64 //最后一个出牌的玩家
	lastHintUid   int64 //最后一个接到提示的玩家

	latestEnter *protocol.PlayerEnterDesk //最新的进入状态

	logger *log.Entry
}

func NewDesk(roomNo room.Number, opts *protocol.DeskOptions, clubId int64) *Desk {
	d := &Desk{
		clubId:  clubId,
		state:   constant.DeskStatusCreate,
		roomNo:  roomNo,
		players: []*Player{},
		group:   nano.NewGroup(uuid.New()),
		die:     make(chan struct{}),

		wonPlayers:   map[int64]bool{},
		isNewRound:   true,
		isFirstRound: true,
		knownTiles:   map[int]int{},
		scoreChanges: map[int64][]*scoreChangeInfo{},
		opts:         opts,
		bankerTurn:   turnUnknown,
		roundStats:   make(history.RoundStats),
		matchStats:   make(history.MatchStats),

		prepare: newPrepareContext(),
		dice:    newDice(),

		logger: log.WithField(fieldDesk, roomNo),
	}

	d.dissolve = newDissolveContext(d)

	return d
}

// 玩家数量
func (d *Desk) totalPlayerCount() int {
	return d.opts.Mode
}

// 麻将数量
func (d *Desk) totalTileCount() int {
	if d.opts.Mode == 4 {
		return 108
	} else {
		return 72
	}
}

func (d *Desk) save() error {
	var name3 string
	var player3 int64
	if d.opts.Mode == ModeFours {
		name3 = d.players[3].name
		player3 = d.players[3].Uid()
	}
	// save to database
	desk := &model.Desk{
		ClubId:      d.clubId,
		Creator:     d.creator,
		CreatedAt:   time.Now().Unix(),
		Mode:        d.opts.Mode,
		DeskNo:      string(d.roomNo),
		Player0:     d.players[0].Uid(),
		Player1:     d.players[1].Uid(),
		Player2:     d.players[2].Uid(),
		Player3:     player3,
		PlayerName0: d.players[0].name,
		PlayerName1: d.players[1].name,
		PlayerName2: d.players[2].name,
		PlayerName3: name3,
		Round:       d.opts.MaxRound, //最多局数
	}

	if data, err := json.Marshal(d.opts); err == nil {
		desk.Extras = string(data)
	}

	d.logger.Infof("保存房间数据, 创建时间: %d", desk.CreatedAt)

	// TODO: 改成异步
	if err := db.InsertDesk(desk); err != nil {
		return err
	}

	d.deskID = desk.Id
	return nil
}

// 如果是重新进入 isReJoin: true
func (d *Desk) playerJoin(s *session.Session, isReJoin bool) error {
	uid := s.UID()
	var (
		p   *Player
		err error
	)

	if isReJoin {
		d.dissolve.updateOnlineStatus(uid, true)
		p, err = d.playerWithId(uid)
		if err != nil {
			d.logger.Errorf("玩家: %d重新加入房间, 但是没有找到玩家在房间中的数据", uid)
			return err
		}

		// 加入分组
		d.group.Add(s)
	} else {
		exists := false
		for _, p := range d.players {
			if p.Uid() == uid {
				exists = true
				p.logger.Warn("玩家已经在房间中")
				break
			}
		}
		if !exists {
			p = s.Value(kCurPlayer).(*Player)
			d.players = append(d.players, p)
			for i, p := range d.players {
				p.setDesk(d, i)
			}
			d.roundStats[uid] = &history.Record{}
		}
	}

	return nil
}

func (d *Desk) syncDeskStatus() {
	d.latestEnter = &protocol.PlayerEnterDesk{Data: []protocol.EnterDeskInfo{}}
	for i, p := range d.players {
		uid := p.Uid()
		d.latestEnter.Data = append(d.latestEnter.Data, protocol.EnterDeskInfo{
			DeskPos:  i,
			Uid:      uid,
			Nickname: p.name,
			IsReady:  d.prepare.isReady(uid),
			Sex:      p.sex,
			IsExit:   false,
			HeadUrl:  p.head,
			Score:    p.score,
			IP:       p.ip,
			Offline:  !d.dissolve.isOnline(uid),
		})
	}
	d.group.Broadcast("onPlayerEnter", d.latestEnter)
}

func (d *Desk) checkStart() {
	s := d.status()
	if (s != constant.DeskStatusCreate) && (s != constant.DeskStatusCleaned) {
		d.logger.Infof("当前房间状态不对，不能开始游戏，当前状态=%s", s.String())
		return
	}

	if count, num := len(d.players), d.totalPlayerCount(); count < num {
		d.logger.Infof("当前房间玩家数量不足，不能开始游戏，当前玩家=%d, 最低数量=%d", count, num)
		return
	}
	for _, p := range d.players { /**/
		if uid := p.Uid(); !d.prepare.isReady(uid) {
			p.logger.Info("玩家未准备")
			return
		}
	}

	d.start()
}

func (d *Desk) scoreChangeForUid(uid int64, sc *scoreChangeInfo) {
	d.logger.Debugf("新增流水: UID=%d, 流水信息=%s", uid, sc.String())
	if _, ok := d.scoreChanges[uid]; !ok {
		d.scoreChanges[uid] = []*scoreChangeInfo{sc}
	} else {
		d.scoreChanges[uid] = append(d.scoreChanges[uid], sc)
	}
}

// 计算胡牌, 下雨, 刮风的分数
func (d *Desk) roundScoreForUid(uid int64) (huScore int, fengScore int, yuScore int) {
	sc, ok := d.scoreChanges[uid]
	if !ok {
		return
	}

	for _, s := range sc {
		d.logger.Debugf("玩家流水: UID=%d, 流水信息=%s", uid, s.String())
		switch s.typ {
		case ScoreChangeTypeHu:
			huScore += s.score
		case ScoreChangeTypeAnGang:
			yuScore += s.score
		case ScoreChangeTypeBaGang:
			fengScore += s.score
		}
	}

	d.logger.Debugf("单局积分: UID=%d, 胡分=%d, 刮风=%d, 下雨=%d", uid, huScore, fengScore, yuScore)
	return
}

// 剩余麻将数量
func (d *Desk) remainTileCount() int {
	return d.totalTileCount() - d.nextTileIndex
}

// 是否是最后一张麻将(先摸后check)
func (d *Desk) noMoreTile() bool {
	return d.remainTileCount() == 0
}

func (d *Desk) title() string {
	return strings.TrimSpace(fmt.Sprintf("房号: %s 局数: %d/%d", d.roomNo, d.round, d.opts.MaxRound))
}

// 描述, 参数表示是否显示额外选项
func (d *Desk) desc(detail bool) string {
	desc := []string{}
	zimo := "自摸加番"
	opts := d.opts
	if opts.Zimo == "di" {
		zimo = "自摸加底"
	}
	desc = append(desc, zimo)
	desc = append(desc, fmt.Sprintf("%d番封顶", d.opts.MaxFan))

	if detail {
		if opts.Pinghu && opts.Mode == ModeTrios {
			desc = append(desc, "点炮可平胡")
		}
		if opts.Menqing {
			desc = append(desc, "门清中张")
		}
		if opts.Jiaxin {
			desc = append(desc, "夹心五")
		}
		if opts.Pengpeng {
			desc = append(desc, "碰碰胡两番")
		}
		if opts.Jiangdui {
			desc = append(desc, "将对")
		}
		if opts.Yaojiu {
			desc = append(desc, "全幺九")
		}
	}

	return strings.Join(desc, " ")
}

// 牌桌开始, 此方法只在开桌时执行, 非并行
func (d *Desk) start() {
	d.round++
	d.setStatus(constant.DeskStatusDuanPai)

	var (
		totalPlayerCount = d.totalPlayerCount() // 玩家数量
		totalTileCount   = d.totalTileCount()   // 麻将数量
	)

	//第一局,随机庄,以后每局的庄家是上一局第一个和牌者或者点双响炮者
	if d.isFirstRound {
		d.isFirstRound = false
		d.loseCoin()
		d.bankerTurn = rand.Intn(totalPlayerCount)

		//只有第一局才创建桌子
		if err := d.save(); err != nil {
			d.logger.Error(err)
		}
	}
	d.curTurn = d.bankerTurn
	// 桌面基本信息
	basic := &protocol.DeskBasicInfo{
		DeskID: d.roomNo.String(),
		Title:  d.title(),
		Desc:   d.desc(true),
		Mode:   d.opts.Mode,
	}

	d.group.Broadcast("onDeskBasicInfo", basic)
	allTiles := mahjong.New(totalTileCount)
	d.logger.Debugf("麻将数量=%d, 玩家数量=%d, 所有麻将=%v", totalTileCount, totalPlayerCount, allTiles)

	info := make([]protocol.DuanPaiInfo, totalPlayerCount)
	for i, p := range d.players {
		info[i] = protocol.DuanPaiInfo{
			Uid:    p.Uid(),
			OnHand: make([]int, 13),
		}
		nextIndex := (i + 1) * 13
		copy(info[i].OnHand, allTiles[i*13:nextIndex])
		d.nextTileIndex = nextIndex
	}
	//庄家多一张牌
	info[d.bankerTurn].OnHand = append(info[d.bankerTurn].OnHand, allTiles[d.nextTileIndex])
	d.nextTileIndex++

	d.allTiles = make(mahjong.Mahjong, len(allTiles))
	for i, id := range allTiles {
		tile := mahjong.TileFromID(id)
		d.allTiles[i] = tile
		if i < d.nextTileIndex {
			d.knownTiles[tile.Index]++
		}
	}

	d.logger.Debugf("游戏开局, 麻将数量=%d 所有麻将: %v", len(d.allTiles), d.allTiles)

	for turn, player := range d.players {
		player.duanPai(info[turn].OnHand)
	}

	// 骰子
	d.dice.random()
	duan := &protocol.DuanPai{
		MarkerID:    info[d.bankerTurn].Uid, //庄的账号ID
		Dice1:       d.dice.dice1,
		Dice2:       d.dice.dice2,
		AccountInfo: info,
	}
	d.group.Broadcast("onDuanPai", duan)

	//d.bankerTurn = turnUnknown //使用完毕,清空以便下一局使用
	name4 := "/"
	if len(d.players) > 3 {
		name4 = d.players[3].name
	}
	d.snapshot = history.New(
		d.deskID,
		d.opts.Mode,
		d.players[0].name,
		d.players[1].name,
		d.players[2].name,
		name4,
		basic,
		d.latestEnter,
		duan,
	)
}

func (d *Desk) qiPaiFinished(uid int64) error {
	if d.status() > constant.DeskStatusDuanPai {
		d.logger.Debugf("当前牌桌状态: %s", d.status().String())
		return errutil.ErrIllegalDeskStatus
	}

	d.prepare.sorted(uid)

	// 等待所有人齐牌
	for _, p := range d.players {
		if !d.prepare.isSorted(p.Uid()) {
			return nil
		}
	}

	d.setStatus(constant.DeskStatusQiPai)

	// 三人不需要定缺
	if d.opts.Mode == ModeTrios {
		go d.play()
	} else {
		for _, p := range d.players {
			que := p.selectDefaultQue()
			p.session.Push("onDingQueHint", protocol.DingQue{que})
		}
	}
	return nil
}

// 定缺
func (d *Desk) dingQue(p *Player, que int) {
	p.ctx.Que = que
	p.logger.Infof("玩家定缺，缺=%d", que)

	// 等待所有人齐牌
	for _, p := range d.players {
		if p.ctx.Que < 1 {
			return
		}
	}

	// 通知所有客户端
	ques := make([]protocol.QueItem, d.totalPlayerCount())
	for i, p := range d.players {
		ques[i] = protocol.QueItem{Uid: p.Uid(), Que: p.ctx.Que}
	}

	d.group.Broadcast("onDingQue", ques)

	go d.play()
}

func (d *Desk) nextTurn() {
	d.curTurn++
	d.curTurn = d.curTurn % d.totalPlayerCount()
}

func (d *Desk) nextTile() *mahjong.Tile {
	tile := d.allTiles[d.nextTileIndex]
	d.nextTileIndex++
	d.knownTiles[tile.Index]++
	if d.knownTiles[tile.Index] > 4 {
		d.logger.Errorf("麻将数量错误, 花色: %s, 已有数量: %d", tile, d.knownTiles[tile.Index])
		d.logger.Debugf("牌桌底牌: %+v", d.allTiles)
	}
	return tile
}

func (d *Desk) isRoundOver() bool {

	//中/终断表示本局结束
	s := d.status()
	if s == constant.DeskStatusInterruption || s == constant.DeskStatusDestory {
		return true
	}

	if d.noMoreTile() {
		return true
	}

	//只剩下一个人没有和牌结算
	return len(d.wonPlayers) == d.totalPlayerCount()-1
}

// 循环中的核心逻辑
// 1. 摸牌
// 2. 检查自扣/暗杠/巴杠
// 3. 打牌
// 4. 检查是否有玩家要碰杠胡
func (d *Desk) play() {
	defer func() {
		if err := recover(); err != nil {
			d.logger.Errorf("Error=%v", err)
			println(stack())
		}
	}()

	d.setStatus(constant.DeskStatusPlaying)
	d.logger.Debug("开始游戏")

	curPlayer := d.players[d.curTurn] //当前出牌玩家,初始为庄家

MAIN_LOOP:
	for !d.isRoundOver() {
		// 切换到下一个玩家
		if !d.isNewRound {
			d.nextTurn()
			curPlayer = d.currentPlayer()
		}

		// 跳过胡牌玩家
		if d.wonPlayers[curPlayer.Uid()] {
			continue
		}

		// 会goto到GANG共有3种情况:
		// 1. 庄家开局后暗杠
		// 2. 玩家暗杠
		// 3. 玩家巴杠
	GANG:
		curPlayer = d.currentPlayer()
		// 1. 如果不是首轮, 上一张牌, 首轮时, 庄家已有14张牌
		if !d.isNewRound {
			curPlayer.moPai()
		}

		//===========================================================
		// 2. 检查自己是否暗杠或者刮风/胡牌
		//===========================================================
		action, tid := curPlayer.doCheckHandTiles(d.isNewRound)
		//  tid　< 0 标识房间解散, just break
		if tid == deskDissolved {
			break MAIN_LOOP
		}

		if d.isNewRound {
			d.isNewRound = false
		}

		// 检查玩家操作
		switch action {
		case protocol.OptypeHu:
			d.wonPlayers[curPlayer.Uid()] = true
			continue

		case protocol.OptypeBaGang:
			if q, dissolve := d.qiangGang(tid); q {
				//有人抢杠和牌, 下一个玩家继续
				continue
			} else if dissolve {
				// 房间解散
				break MAIN_LOOP
			} else {
				//无人抢杠,让当前玩家继续执行杠
				//明杠数
				d.roundStats[curPlayer.Uid()].MingGangNum++

				loser := d.allLosers(curPlayer)
				curPlayer.gangBySelf(tid, false, loser)

				curPlayer.logger.Infof("自摸巴明杠 牌=%d 当前状态=%d", tid, curPlayer.ctx.PrevOp)
				for uid, stats := range d.roundStats {
					d.logger.Infof("自摸巴明杠: 玩家=%d 明杠=%d, 暗杠=%d", uid, stats.MingGangNum, stats.AnGangNum)
				}

				goto GANG
			}

		case protocol.OptypeGang:
			//暗杠数
			d.roundStats[curPlayer.Uid()].AnGangNum++

			loser := d.allLosers(curPlayer)
			curPlayer.gangBySelf(tid, true, loser)

			curPlayer.logger.Info("自摸暗杠")
			for uid, stats := range d.roundStats {
				d.logger.Infof("自摸暗杠: 玩家=%d 明杠=%d 暗杠=%d\n", uid, stats.MingGangNum, stats.AnGangNum)
			}

			goto GANG
		}

	PENG:
		curPlayer = d.currentPlayer()
		//===========================================================
		// 检查有无其他玩家要当前玩家出的牌
		//===========================================================
		// 3. 出牌
		tid = curPlayer.chuPai()
		//  tid　< 0 标识房间解散, just break
		if tid == deskDissolved {
			break MAIN_LOOP
		}

		d.lastTileId = tid
		d.lastChuPaiUid = curPlayer.Uid()
		curPlayer.chupai = append(curPlayer.chupai, mahjong.TileFromID(tid))
		curPlayer.ctx.LastDiscardId = tid

		// 4. 检查是否有玩家要这张牌
		typ := d.chiPai(tid)
		if typ == deskDissolved {
			break MAIN_LOOP
		}

		// 记录出牌
		if typ == protocol.OptypeGang || typ == protocol.OptypePeng || typ == protocol.OptypeHu {
			curPlayer.chupai = curPlayer.chupai[:len(curPlayer.chupai)-1]
		}

		//出牌玩家未点杠上炮,清空其前一次的杠牌操作
		//if typ != protocol.OptypeHu && curPlayer.ctx.PrevOp == protocol.OptypeGang {
		curPlayer.ctx.SetPrevOp(protocol.OptypeChu)
		//}

		// 杠牌
		if typ == protocol.OptypeGang {

			//此时currentPlayer已经更新
			gangPlayer := d.currentPlayer()

			//gangPlayer.ctx.SetPrevOp(protocol.OptypeGang)

			//明杠数
			d.roundStats[gangPlayer.Uid()].MingGangNum++

			gangPlayer.logger.Info("点明杠")
			for uid, stats := range d.roundStats {
				d.logger.Infof("点明杠: 玩家=%d 明杠=%d 暗杠=%d", uid, stats.MingGangNum, stats.AnGangNum)
			}

			goto GANG
		}

		// 碰牌
		if typ == protocol.OptypePeng {
			goto PENG
		}
	}

	if d.status() == constant.DeskStatusDestory {
		d.logger.Info("已经销毁(三人都离线或解散)")
		return
	}

	if d.status() != constant.DeskStatusInterruption {
		d.setStatus(constant.DeskStatusRoundOver)
	}

	d.roundOver()
}

func (d *Desk) currentPlayer() *Player {
	return d.players[d.curTurn]
}

func (d *Desk) allLosers(win *Player) []int64 {
	loser := []int64{}
	for _, u := range d.players {
		uid := u.Uid()
		//跳过自己与已和玩家
		if uid == win.Uid() || d.wonPlayers[uid] {
			continue
		}
		loser = append(loser, uid)
	}
	return loser
}

// 转雨，需要考虑一下集中清空
// 1. 普通转雨
// 2. 点杠后，对方杠上炮，转雨给自己
// 3. 暗杠或者巴杠后，一炮双向，另外两家相互转雨
func (d *Desk) zhuanYu(huUid, chuUid int64) {
	// 转雨: 杠牌后, 如果点炮, 则杠牌的钱转给胡牌的人
	if changes, ok := d.scoreChanges[chuUid]; ok {
		// 找到最后一个下雨或者刮风的分值改变记录
		cl := len(changes)
		lastChange := changes[len(changes)-1]
		for i := cl; i > 0; i-- {
			if c := changes[i-1]; c.typ == ScoreChangeTypeAnGang || c.typ == ScoreChangeTypeBaGang {
				lastChange = c
				break
			}
		}
		lastTileId := lastChange.tileID

		// 清除赢牌人的积分, 刮风下雨有可能是赢了两家人
		for i := range changes {
			c := changes[i]
			if c.score > 0 && c.tileID == lastTileId && (c.typ == ScoreChangeTypeAnGang || c.typ == ScoreChangeTypeBaGang) {
				c.score = 0
			}
			d.logger.Debugf("玩家ID=%d 流水=%s", chuUid, c.String())
		}

		// 修改输牌人把分输给谁了
		for _uid, _changes := range d.scoreChanges {
			// 前面已经处理了出牌人的情况
			if _uid == chuUid {
				continue
			}

			// FIXED: 自己杠，然后对方在点炮，转雨转回自己
			cl := len(_changes)
			for j := 0; j < cl; j++ {
				c := _changes[j]
				d.logger.Debugf("玩家ID=%d 流水=%s", _uid, c.String())
				if c.uid == chuUid && c.tileID == lastTileId && (c.typ == ScoreChangeTypeAnGang || c.typ == ScoreChangeTypeBaGang) {
					// 清除输掉的积分
					c.score = 0

					// 不需要转雨给自己
					if _uid == huUid {
						continue
					}

					score := 1
					if c.typ == ScoreChangeTypeAnGang {
						score = 2
					}

					// 转雨给胡牌的人
					d.scoreChangeForUid(_uid, &scoreChangeInfo{
						score:  -score, //之前是输分, 为负, 现在改为正
						uid:    huUid,  //谁输的
						typ:    c.typ,
						tileID: lastTileId,
					})

					d.scoreChangeForUid(huUid, &scoreChangeInfo{
						score:  score, //之前是输分, 为负, 现在改为正
						uid:    _uid,  //谁输的
						typ:    c.typ,
						tileID: lastTileId,
					})
				}
			}
		}
	} else {
		panic("玩家杠牌, 但是没有杠牌记录")
	}
}

func (d *Desk) chiPai(tileId int) int {
	//出牌的玩家
	chuPlayer := d.currentPlayer()
	curTile := mahjong.TileFromID(tileId)
	playerCount := d.totalPlayerCount()

	// 某个方位可用的操作
	type turnOp struct {
		turn int
		op   []protocol.Op
	}

	// 检查有没有玩家要碰、杠、胡
	checkTurn := d.curTurn
	winOps := map[int]bool{}                                // 某个方位是否可以胡这张牌
	pgOps := turnOp{turn: illegalTurn, op: []protocol.Op{}} // 牌桌同时只可能有一个人能碰杠
	for {
		// 从下家开始检查, 一直检查到自己
		checkTurn++
		checkTurn = checkTurn % playerCount
		if checkTurn == d.curTurn {
			break
		}

		// 当前检查玩家
		checkPlayer := d.players[checkTurn]

		//已经和牌 跳过
		if d.wonPlayers[checkPlayer.Uid()] {
			continue
		}

		ops := checkPlayer.checkChi(tileId, chuPlayer)
		if len(ops) == 0 {
			continue
		}

		for _, op := range ops {
			switch op.Type {
			case protocol.OptypeGang:
				pgOps.turn = checkTurn
				pgOps.op = append(pgOps.op, protocol.Op{Type: protocol.OptypePeng, TileIDs: []int{tileId}})
				//可杠就可碰,如桌面上已经无牌，则不可杠, 但最后一张可以碰
				if !d.noMoreTile() {
					pgOps.op = append(pgOps.op, protocol.Op{Type: protocol.OptypeGang, TileIDs: []int{tileId}})
				}

			case protocol.OptypePeng:
				pgOps.turn = checkTurn
				pgOps.op = []protocol.Op{{Type: protocol.OptypePeng, TileIDs: []int{tileId}}}

			case protocol.OptypeHu:
				// 当前打牌玩家的上家
				preTurn := d.curTurn - 1
				if preTurn < 0 {
					preTurn += playerCount
				}

				// 当前检查吃牌玩家的下家
				nextTurn := checkTurn + 1

				d.logger.Debugf("检查过手胡，CurTurn=%d CheckTurn=%d PreTurn=%d NextTurn=%d, CurTileIndex=%d CurTileID=%d",
					d.curTurn, checkTurn, preTurn, nextTurn, curTile.Index, tileId)

				// 过手胡规则处理
				if chuPlayer.ctx.PrevOp == protocol.OptypeGang || d.noMoreTile() || preTurn == checkTurn {
					// 如果是杠上炮或者海底捞, 允许过手胡
					// 如果是上家可以胡，也直接胡牌
					winOps[checkTurn] = true
				} else {
					// 如果当前打牌和上家打牌一样，则不能点炮
					// 从nextTurn的下一家一直检查到打牌人的上家，如果最后一张打牌和当前牌的index一样
					// 则不允许和牌
					for nextTurn > preTurn {
						preTurn += playerCount
					}
					d.logger.Debugf("-->检查过手胡，CurTurn=%d CheckTurn=%d PreTurn=%d NextTurn=%d, CurTileIndex=%d CurTileID=%d",
						d.curTurn, checkTurn, preTurn, nextTurn, curTile.Index, tileId)

					hasGuo := false
					for i := nextTurn; i <= preTurn; i++ {
						p := d.players[i%playerCount]
						// 已经和牌的玩家不检查
						if d.wonPlayers[p.uid] {
							continue
						}
						// 上一张打掉掉牌
						id := p.ctx.LastDiscardId
						d.logger.Debugf("==>检查过手胡，CurTurn=%d CheckTurn=%d PreTurn=%d NextTurn=%d, Index=%d, CurTileIndex=%d CurTileID=%d LastID=%d",
							d.curTurn, checkTurn, preTurn, nextTurn, i, curTile.Index, tileId, id)
						if id != mahjong.IllegalIndex && mahjong.TileFromID(id).Index == curTile.Index {
							hasGuo = true
							break
						}
					}

					// 没有过手胡
					if !hasGuo {
						winOps[checkTurn] = true
					}
				}
			}
		}
	}

	d.logger.Debugf("可以胡牌的玩家: %+v", winOps)
	d.logger.Debugf("可以碰杠的玩家: %+v", pgOps)

	// 是否有人胡牌，一炮多响
	paoCount := 0

	// 先将提示发过去
	// 如果胡牌的人同时可以碰杠, 将提示一起发送过去, 胡牌优先
	for winTurn := range winOps {
		player := d.players[winTurn]
		hints := []protocol.Op{
			{Type: protocol.OptypeHu, TileIDs: []int{tileId}},
			{Type: protocol.OptypePass},
		}
		if winTurn == pgOps.turn {
			hints = append(hints, pgOps.op...)
		}
		player.hint(hints)
	}

	// TODO: 如果一炮双响的时候, 先通知的玩家还可以碰/杠/玩家选择碰杠, 第二个玩家选择胡牌, 需要仔细推敲
	for winTurn := range winOps {
		player := d.players[winTurn]
		uid := player.Uid()

		// 选择了过
		optype := player.hu(tileId, true)
		if optype == deskDissolved {
			return deskDissolved
		}
		if optype != protocol.OptypeHu {
			continue
		}

		paoCount++

		d.wonPlayers[uid] = true
		d.curTurn = winTurn

		d.logger.Debugf("玩家胡牌: UID=%d 胡=%+v 手牌=%+v 碰杠=%+v",
			uid, mahjong.TileFromID(tileId), player.handTiles(), player.pgTiles())

		player.action(protocol.OptypeHu, []int{tileId})

		player.ctx.NewOtherDiscardID = tileId
		player.onHand = append(player.onHand, mahjong.TileFromID(tileId))
		//fixed: 设置玩家的winTileId
		player.ctx.WinningID = tileId

		//放炮玩家:杠上炮
		if chuPlayer.ctx.PrevOp == protocol.OptypeGang {
			chuUid := chuPlayer.Uid()
			d.logger.Debugf("杠上炮: 胡牌玩家=%d, 放炮玩家=%d", uid, chuUid)
			player.ctx.IsGangShangPao = true

			// 转雨
			d.zhuanYu(uid, chuUid)
		}
		score := player.scoring()

		d.logger.Debugf("玩家胡牌: UID=%d 赢分=%d", player.Uid(), score)

		d.paoPlayer = chuPlayer.Uid()
		chuPlayer.ctx.ResultType = ResultPao
		player.ctx.ResultType = ResultHu

		losers := []Loser{{uid: chuPlayer.Uid(), score: score}}

		d.scoreChangeForHu(player, losers, tileId, protocol.HuTypeDianPao)
	}

	//点炮
	if paoCount > 0 {
		if paoCount == 2 {
			d.setNextRoundBanker(chuPlayer.Uid(), true)
		}
		return protocol.OptypeHu
	}

	if pgOps.turn == illegalTurn {
		return protocol.OptypePass
	}

	// 无人胡牌, 则通知可以碰杠的玩家, 如果玩家可以碰杠胡, 前面已经发送了碰杠提示, 但是未选择胡
	player := d.players[pgOps.turn]
	isPeng, isGang, isDissolve := player.pengOrGang(tileId, pgOps.op, winOps[pgOps.turn])
	if isDissolve {
		return deskDissolved
	}

	d.logger.Debugf("玩家碰杠结果: 碰=%t 杠=%t", isPeng, isGang)
	if !isPeng && !isGang {
		return protocol.OptypePass
	}

	d.curTurn = pgOps.turn

	//点杠只可能是自己手中已有3张的(引风)下雨,即另一种明杠
	if isGang {
		// 客户端显示杠牌流水
		d.scoreChangeForGang(player, []int64{chuPlayer.Uid()}, tileId, true)
		return protocol.OptypeGang
	}

	return protocol.OptypePeng
}

// 检查有没有玩家抢杠, 第二个参数表示房间是否解散
func (d *Desk) qiangGang(tid int) (bool, bool) {
	//出牌的玩家
	chuPlayer := d.currentPlayer()

	// 检查有没有玩家要胡
	nextTurn := d.curTurn
	paoCount := 0

	for {
		// 从下家开始检查, 一直检查到自己
		nextTurn++
		nextTurn = nextTurn % d.totalPlayerCount()

		//已经检查了一轮
		if nextTurn == d.curTurn {
			break
		}
		nextPlayer := d.players[nextTurn]

		//已经和牌 跳过
		if d.wonPlayers[nextPlayer.Uid()] {
			continue
		}

		//不能抢杠和
		if !nextPlayer.checkHu(tid, true) {
			continue
		}

		//可以和,但是不抢杠
		optype := nextPlayer.hu(tid, false)
		if optype == deskDissolved {
			return false, true
		}
		if optype != protocol.OptypeHu {
			continue
		}

		paoCount++
		d.wonPlayers[nextPlayer.Uid()] = true

		d.logger.Debugf("玩家胡牌: UID=%d 胡=%+v 手牌=%+v 碰杠=%+v",
			nextPlayer.Uid(), mahjong.TileFromID(tid), nextPlayer.handTiles(), nextPlayer.pgTiles())

		nextPlayer.action(protocol.OptypeHu, []int{tid})
		nextPlayer.ctx.SetPrevOp(protocol.OptypeHu)
		nextPlayer.ctx.WinningID = tid
		nextPlayer.ctx.IsQiangGangHu = true
		nextPlayer.onHand = append(nextPlayer.onHand, mahjong.TileFromID(tid))

		score := nextPlayer.scoring()
		losers := []Loser{{uid: chuPlayer.Uid(), score: score}}
		d.scoreChangeForHu(nextPlayer, losers, tid, protocol.HuTypeDianPao)

		//切换当前玩家
		d.curTurn = nextPlayer.turn
	}

	//抢杠双响炮
	if paoCount > 0 {
		if paoCount == 2 {
			d.setNextRoundBanker(chuPlayer.Uid(), true)
		}

		//有人抢杠,从杠牌玩家手里移除被抢杠的牌
		mahjong.RemoveId(&(chuPlayer.onHand), tid)
		return true, false
	}

	return false, false
}

func (d *Desk) nobodyWin() bool {
	for _, ok := range d.wonPlayers {
		if ok {
			return false
		}
	}
	return true
}

func (d *Desk) roundOverTilesForPlayer(p *Player) *protocol.HandTilesInfo {
	uid := p.Uid()
	ids := p.handTiles().Ids()
	sps := []int{}

	// fixed: 将胡牌从手牌中移除
	winTileID := -1
	if d.wonPlayers[uid] {
		winTileID = p.ctx.WinningID
		for _, id := range ids {
			if id == winTileID {
				continue
			}
			sps = append(sps, id)
		}
	} else {
		sps = make([]int, len(ids))
		copy(sps, ids)
	}

	// 手牌
	tiles := &protocol.HandTilesInfo{
		Uid:    uid,
		Tiles:  sps,
		HuPai:  winTileID,
		IsTing: d.wonPlayers[uid] || p.isTing(),
	}

	return tiles
}

func (d *Desk) roundOverStatsForPlayer(p *Player) *protocol.RoundStats {
	uid := p.Uid()

	huScore, fengScore, yuScore := d.roundScoreForUid(uid)
	total := huScore + yuScore + fengScore

	d.logger.Debugf("单局结算: 玩家ID=%d huScore=%d yuScore=%d total=%d bannerType=%d",
		uid, huScore, yuScore, total, p.ctx.ResultType)

	stats := &protocol.RoundStats{
		Feng:       fengScore,
		Yu:         yuScore,
		Total:      total,
		FanNum:     p.ctx.Fan,
		BannerType: p.ctx.ResultType,
	}

	return stats
}

func (d *Desk) roundOverHelper() *protocol.RoundOverStats {
	// 游戏结束
	overStats := &protocol.RoundOverStats{
		Round:       fmt.Sprintf("局数: %d/%d", d.round, d.opts.MaxRound),
		Title:       d.desc(false),
		HandTiles:   []*protocol.HandTilesInfo{},
		ScoreChange: []protocol.GameEndScoreChange{},
		Stats:       []*protocol.RoundStats{},
	}

	if d.status() == constant.DeskStatusCleaned {
		return overStats
	}

	playerCount := d.totalPlayerCount()

	// 无叫玩家需要赔付给有叫玩家, 并且无叫的玩家之前所有刮风下雨积分全部清零
	pei := map[int64]struct{}{}
	ting := map[int64]struct{}{}
	for _, p := range d.players {
		uid := p.Uid()
		tiles := d.roundOverTilesForPlayer(p)

		overStats.HandTiles = append(overStats.HandTiles, tiles)

		// 只有一个人没有和牌，不赔叫
		if len(d.wonPlayers) == playerCount-1 {
			continue
		}

		if !d.wonPlayers[uid] {
			// 没有胡牌, 并且没叫
			if !tiles.IsTing {
				pei[uid] = struct{}{}
			} else {
				ting[uid] = struct{}{}
			}
		}
	}

	// 是否需要查叫: 牌桌的牌全部摸完, 并且还有人没胡牌
	// 有人有叫有人没叫
	if len(pei) > 0 {
		// 要赔叫的人, 之前的所有刮风下雨清零
		for uid := range pei {
			if changes, ok := d.scoreChanges[uid]; ok {
				for i := range changes {
					c := changes[i]
					// 清空正的刮风下雨积分(负分是输给别人的, 不能清空)
					if c.score > 0 && (c.typ == ScoreChangeTypeBaGang || c.typ == ScoreChangeTypeAnGang) {
						c.score = 0
					}
				}
			}
		}
		for _, changes := range d.scoreChanges {
			// 清除其他玩家的输分
			for i := range changes {
				change := changes[i]
				winner := change.uid
				// 如果赢家要赔付, 则清0不赔付
				if _, ok := pei[winner]; ok && change.score < 0 && (change.typ == ScoreChangeTypeBaGang || change.typ == ScoreChangeTypeAnGang) {
					change.score = 0
				}
			}
		}

		// 给有叫的人赔叫, 按最大的番数赔
		// 有可能剩下两个人都没叫，不需要赔其他人，只需要清空刮风下雨都积分即可
		for uid := range ting {
			if p, err := d.playerWithId(uid); err == nil {
				score, index := p.maxTingScore()
				losers := []Loser{}
				for loser := range pei {
					losers = append(losers, Loser{loser, score})
				}
				d.scoreChangeForHu(p, losers, index, protocol.HuTypePei)
			}
		}
	}

	// 总结算分数
	for i, p := range d.players {
		uid := p.Uid()
		stats := d.roundOverStatsForPlayer(p)
		total := stats.Total
		p.score += total
		d.roundStats[uid].TotalScore = total

		if !d.wonPlayers[uid] {
			stats.FanNum = mahjong.MeiHu
		} else {
			if stats.FanNum != 0 {
				stats.Desc = strings.Join(p.ctx.Desc, " ")
			}
		}

		sc := protocol.GameEndScoreChange{Uid: uid, Score: total, Remain: p.score}
		overStats.ScoreChange = append(overStats.ScoreChange, sc)
		overStats.Stats = append(overStats.Stats, stats)

		if d.snapshot != nil {
			d.snapshot.SetScoreChangeForTurn(byte(i), total)
		}
	}

	return overStats
}

func (d *Desk) setStatus(s constant.DeskStatus) {
	atomic.StoreInt32((*int32)(&d.state), int32(s))
}

func (d *Desk) status() constant.DeskStatus {
	return constant.DeskStatus(atomic.LoadInt32((*int32)(&d.state)))
}

func (d *Desk) roundOver() {
	stats := d.roundOverHelper()
	status := d.status()

	//只有正常结束的牌局才需要回放
	//只有在已经开始本局或者正常结束时才需要缓存单局统计
	if status == constant.DeskStatusRoundOver {
		d.snapshot.SetEndStats(stats)
		d.snapshot.Save()
		d.matchStats.Push(d.roundStats)
	}

	//满场
	isMaxRound := d.round >= uint32(d.opts.MaxRound) && status == constant.DeskStatusRoundOver

	d.logger.Debugf("本轮游戏结束, 状态=%s 结算数据=%#v", status.String(), stats)
	//round over
	if status == constant.DeskStatusRoundOver && !isMaxRound {
		d.group.Broadcast("onRoundEnd", stats)
		d.clean()
	} else {
		//最后一局以及中断统计的GameEnd与场结算一起发送
		d.finalSettlement(isMaxRound, stats)
	}

	/*
		walkaround:
		1. A切后台,开始定时器, B,C弹出"申请解散对话框"
		2. B申请解散
		3. B,C Ready(正常情况此处在Ready前，有一个 C,允许\拒绝的消息)
		4. A切回前台,开始游戏,此时A引发的定时器尚未撤销的bug
	*/
	d.dissolve.stop()
}

func (d *Desk) clean() {
	d.state = constant.DeskStatusCleaned
	d.isNewRound = true
	d.snapshot = nil
	d.isFirstRound = false

	d.knownTiles = map[int]int{}
	d.allTiles = mahjong.Mahjong{}
	d.nextTileIndex = 0
	d.wonPlayers = map[int64]bool{}

	d.lastChuPaiUid = illegalTile
	d.lastHintUid = illegalTile

	d.scoreChanges = map[int64][]*scoreChangeInfo{}
	d.paoPlayer = -1

	d.isMakerSet = false

	d.dissolve.reset()
	d.prepare.reset()

	//重置玩家状态
	for _, p := range d.players {
		d.roundStats[p.Uid()] = &history.Record{}
		p.reset()
	}
}

func (d *Desk) finalSettlement(isNormalFinished bool, ge *protocol.RoundOverStats) {
	d.logger.Debugf("本场游戏结束, 最后一局结算数据: %#v", ge)
	stats := d.matchStats.Result()

	f := func() []protocol.MatchStats {
		mss := make([]protocol.MatchStats, 0)

		for _, p := range d.players {

			uid := p.Uid()
			var ms protocol.MatchStats

			if uid == d.creator {
				ms.IsCreator = true
			}

			if stats[p.Uid()] != nil {
				ms.TotalScore = stats[p.Uid()].TotalScore
				ms.ZiMoNum = stats[p.Uid()].ZiMoNum
				ms.HuNum = stats[p.Uid()].HuNum
				ms.PaoNum = stats[p.Uid()].PaoNum
				ms.AnGangNum = stats[p.Uid()].AnGangNum
				ms.MingGangNum = stats[p.Uid()].MingGangNum
			}

			ms.Uid = uid
			ms.Account = p.name

			mss = append(mss, ms)
		}

		//炮王
		pws := map[int][]int{}

		//大赢家
		dyjs := map[int][]int{}

		for i := 0; i < len(mss); i++ {
			//pws[mss[i].PaoNum] = i
			//dyjs[mss[i].TotalScore] = i

			if _, ok := pws[mss[i].PaoNum]; !ok {
				pws[mss[i].PaoNum] = []int{}
			}
			pws[mss[i].PaoNum] = append(pws[mss[i].PaoNum], i)

			if _, ok := dyjs[mss[i].TotalScore]; !ok {
				dyjs[mss[i].TotalScore] = []int{}
			}
			dyjs[mss[i].TotalScore] = append(dyjs[mss[i].TotalScore], i)

		}

		//排除所有变化为0的非正常记录
		delete(dyjs, 0)
		delete(pws, 0)

		for k, v := range dyjs {
			for _, v1 := range v {
				d.logger.Debugf("大赢家: K=%d V=%v", k, v1)
			}
		}

		for k, v := range pws {
			for _, v1 := range v {
				d.logger.Debugf("炮王: K=%d V=%v", k, v1)
			}
		}

		chooser := func(m map[int][]int) []int {
			if len(m) == 0 {
				return nil
			}

			//1.提取key(TotalScore, PaoNum)
			keys := make([]int, 0)
			for k := range m {
				keys = append(keys, k)
			}

			//2.基于key排序
			sort.Slice(keys, func(i, j int) bool {
				return keys[i] > keys[j]
			})

			//3.选出炮王与大赢家的index(可能有多个）
			return m[keys[0]]
		}

		pwKeys := chooser(pws)
		dyjKeys := chooser(dyjs)

		for _, i := range pwKeys {
			mss[i].IsPaoWang = true
		}
		for _, i := range dyjKeys {
			mss[i].IsBigWinner = true
		}

		return mss
	}

	mss := f()
	ddr := &protocol.DestroyDeskResponse{
		MatchStats:       mss,
		Title:            d.title(),
		RoundStats:       ge,
		IsNormalFinished: isNormalFinished,
	}

	//发送单场统计
	err := d.group.Broadcast("onGameEnd", ddr)
	if err != nil {
		log.Error(err)
	}

	//桌子解散,更新桌面信息
	desk := &model.Desk{
		Id:      d.deskID,
		Round:   d.matchStats.Round(),
		ClubId:  d.clubId,
		Creator: d.creator,
		DeskNo:  d.roomNo.String(),
	}

	for i := range d.players {
		p := d.players[i]
		uid := p.Uid()
		score := 0
		if r, ok := stats[uid]; ok {
			score = r.TotalScore
		}
		switch i {
		case 0:
			desk.Player0, desk.ScoreChange0, desk.PlayerName0 = uid, score, p.name
		case 1:
			desk.Player1, desk.ScoreChange1, desk.PlayerName1 = uid, score, p.name
		case 2:
			desk.Player2, desk.ScoreChange2, desk.PlayerName2 = uid, score, p.name
		case 3:
			desk.Player3, desk.ScoreChange3, desk.PlayerName3 = uid, score, p.name
		}
	}

	d.destroy()

	// 数据库异步更新
	async.Run(func() {
		if err = db.UpdateDesk(desk); err != nil {
			log.Error(err)
		}
	})
}

func (d *Desk) isDestroy() bool {
	return d.status() == constant.DeskStatusDestory
}

// 摧毁桌子
func (d *Desk) destroy() {
	if d.status() == constant.DeskStatusDestory {
		d.logger.Info("桌子已经解散")
		return
	}

	close(d.die)

	// 标记为销毁
	d.setStatus(constant.DeskStatusDestory)

	d.logger.Info("销毁房间")
	for i := range d.players {
		p := d.players[i]
		d.logger.Debugf("销毁房间，清除玩家%d数据", p.Uid())
		p.reset()
		p.desk = nil
		p.score = 1000
		p.turn = 0
		p.logger = log.WithField(fieldPlayer, p.uid)
		d.players[i] = nil
	}

	// 释放desk资源
	d.group.Close()
	d.prepare.reset()
	d.dissolve.reset()
	d.wonPlayers = nil
	d.snapshot = nil
	d.knownTiles = nil
	d.matchStats = nil
	d.scoreChanges = nil
	d.roundStats = nil

	//删除桌子
	scheduler.PushTask(func() {
		defaultDeskManager.setDesk(d.roomNo, nil)
	})
}

func (d *Desk) scoreChangeHelper(winner int64, losers []Loser, typ ScoreChangeType, tileID int) {
	//向赢/输者队列添加信息
	for _, loser := range losers {
		d.scoreChangeForUid(winner, &scoreChangeInfo{
			score:  loser.score, //赢了多少(正分)
			uid:    loser.uid,   //谁输的
			typ:    typ,
			tileID: tileID,
		})

		d.scoreChangeForUid(loser.uid, &scoreChangeInfo{
			score:  -loser.score, //输了多少(负分)
			uid:    winner,       //谁赢的
			typ:    typ,
			tileID: tileID,
		})
	}
}
func (d *Desk) scoreChangeForGang(winner *Player, loserPlayers []int64, tileID int, isXiaYu bool) {
	gsc := &protocol.GangPaiScoreChange{
		IsXiaYu: isXiaYu,
	}

	score := 1
	typ := ScoreChangeTypeBaGang
	if isXiaYu {
		score = score * 2
		typ = ScoreChangeTypeAnGang
	}

	winScore := score * len(loserPlayers)

	gsc.Changes = append(gsc.Changes, protocol.ScoreInfo{
		Uid:   winner.Uid(),
		Score: winScore,
	})

	var losers []Loser
	for _, uid := range loserPlayers {
		gsc.Changes = append(gsc.Changes, protocol.ScoreInfo{
			Uid:   uid,
			Score: -score,
		})

		losers = append(losers, Loser{
			uid:   uid,
			score: score,
		})

	}

	d.group.Broadcast("onGangScoreChange", gsc)
	d.scoreChangeHelper(winner.Uid(), losers, typ, tileID)
	d.snapshot.PushGangScoreChange(gsc)
}

func (d *Desk) maxScore() int {
	return 1 << uint(d.opts.MaxFan)
}

// 番数过滤
func (d *Desk) maxScoreCutter(point int) int {
	max := d.maxScore()
	if point > max {
		return max
	}
	return point
}

func (d *Desk) scoreChangeForHu(winner *Player, losers []Loser, tileID int, huType protocol.HuPaiType) {
	for _, l := range losers {
		d.logger.Debugf("scoreChangeForHu 赢家=%d 输家=%d 分值=%d 类型=%d 牌=%s",
			winner.Uid(), l.uid, l.score, huType, mahjong.IndexFromID(tileID))
	}

	var winUid = winner.Uid()

	//第一个和的玩家则为下一局的庄, 如果是双响炮则退回上一层处理
	d.setNextRoundBanker(winUid, false)

	//和牌数
	d.roundStats[winUid].HuNum++

	if huType == protocol.HuTypeZiMo {
		d.roundStats[winUid].ZiMoNum++
		winner.ctx.ResultType = ResultZiMo
	} else if len(losers) > 0 {
		for i := range losers {
			loser := losers[i].uid
			d.roundStats[loser].PaoNum++
			if p, err := d.playerWithId(loser); err == nil {
				if huType == protocol.HuTypePei {
					p.ctx.ResultType = ResultPei
				} else {
					winner.ctx.ResultType = ResultHu
					p.ctx.ResultType = ResultPao
				}
			}
		}
	} else {
		panic("empty losers")
	}

	// 极品用-1表示
	if winner.ctx.Fan >= winner.desk.opts.MaxFan {
		winner.ctx.Fan = mahjong.MaxFan
	}

	hsc := &protocol.HuInfo{
		Uid:         winUid,
		HuPaiType:   huType,
		ScoreChange: []protocol.ScoreInfo{},
	}

	isJiaDi := false
	if huType == protocol.HuTypeZiMo && d.opts.Zimo == "di" {
		isJiaDi = true
	}
	for i := 0; i < len(losers); i++ {
		losers[i].score = d.maxScoreCutter(losers[i].score)
		// 自摸加底
		if isJiaDi {
			losers[i].score += 1
		}
		// 计算总分
		hsc.TotalWinScore += losers[i].score
		hsc.ScoreChange = append(hsc.ScoreChange, protocol.ScoreInfo{
			Uid: losers[i].uid,
			//所有的碰杠输赢
			Score: -losers[i].score,
		})
	}

	// 分值变化统计
	d.scoreChangeHelper(winUid, losers, ScoreChangeTypeHu, tileID)

	d.snapshot.PushHuScoreChange(hsc)
	d.group.Broadcast("onHuScoreChange", hsc)
}

//桌上的最后一张牌
func (d *Desk) lastTile() *mahjong.Tile {
	return d.allTiles[d.totalTileCount()-1]
}

func (d *Desk) onPlayerExit(s *session.Session, isDisconnect bool) {
	uid := s.UID()
	d.group.Leave(s)
	if isDisconnect {
		d.dissolve.updateOnlineStatus(uid, false)
	} else {
		restPlayers := []*Player{}
		for _, p := range d.players {
			if p.Uid() != uid {
				restPlayers = append(restPlayers, p)
			} else {
				p.reset()
				p.desk = nil
				p.score = 1000
				p.turn = 0
			}
		}
		d.players = restPlayers
	}

	//如果桌上已无玩家, destroy it
	if d.creator == uid && !isDisconnect {
		//if d.dissolve.offlineCount() == len(d.players) || (d.creator == uid && !isDisconnect) {
		d.logger.Info("所有玩家下线或房主主动解散房间")
		if d.dissolve.isDissolving() {
			d.dissolve.stop()
		}
		d.destroy()

		// 数据库异步更新
		async.Run(func() {
			desk := &model.Desk{
				Id:    d.deskID,
				Round: 0,
			}
			if err := db.UpdateDesk(desk); err != nil {
				log.Error(err)
			}
		})
	}
}

func (d *Desk) playerWithId(uid int64) (*Player, error) {
	for _, p := range d.players {
		if p.Uid() == uid {
			return p, nil
		}
	}

	return nil, errutil.ErrPlayerNotFound
}

func (d *Desk) setNextRoundBanker(uid int64, override bool) {
	// 如果已经设置了庄家，如果一炮双响则重新设置庄家
	if d.isMakerSet && !override {
		return
	}
	for i, p := range d.players {
		if p.Uid() == uid {
			d.bankerTurn = i
			break
		}
	}
	d.isMakerSet = true
}

// 申请解散
func (d *Desk) applyDissolve(uid int64) {
	if d.dissolve.isDissolving() {
		d.logger.Info("已经有人申请解散")
		return
	}

	d.dissolve.start(applyDissolveRestTime)

	// 重置解散状态
	d.dissolve.reset()
	d.dissolve.setUidStatus(uid, true, ApplyDissolve)

	d.group.Broadcast("onDissolveAgreement", &protocol.DissolveResponse{
		DissolveUid:    uid,
		DissolveStatus: d.collectDissolveStatus(),
		RestTime:       applyDissolveRestTime,
	})
}

// 收集本桌玩家的解散状态
func (d *Desk) collectDissolveStatus() []protocol.DissolveStatusItem {
	list := []protocol.DissolveStatusItem{}
	for i, p := range d.players {
		uid := p.Uid()
		status := Waiting
		// 如果同意解散后离线，则解散状态为同意解散，而非离线，如果解散状态为等待中，离线则改为离线
		paused := d.dissolve.pause[uid]
		if s, ok := d.dissolve.desc[uid]; paused && (!ok || s == Waiting) {
			d.dissolve.desc[uid] = Offline
		} else if s == Offline && !paused {
			d.dissolve.desc[uid] = Waiting
		}

		if s, ok := d.dissolve.desc[uid]; ok {
			status = s
		}

		list = append(list, protocol.DissolveStatusItem{
			DeskPos: i + 1,
			Status:  status,
		})
	}
	return list
}

func (d *Desk) onPlayerReJoin(s *session.Session) error {
	// 同步房间基本信息
	basic := &protocol.DeskBasicInfo{
		DeskID: d.roomNo.String(),
		Title:  d.title(),
		Desc:   d.desc(true),
	}
	if err := s.Push("onDeskBasicInfo", basic); err != nil {
		log.Error(err.Error())
		return err
	}

	// 同步所有玩家数据
	enter := &protocol.PlayerEnterDesk{Data: []protocol.EnterDeskInfo{}}
	for i, p := range d.players {
		uid := p.Uid()
		enter.Data = append(enter.Data, protocol.EnterDeskInfo{
			DeskPos:  i,
			Uid:      uid,
			Nickname: p.name,
			IsReady:  d.prepare.isReady(uid),
			Sex:      p.sex,
			IsExit:   false,
			HeadUrl:  p.head,
			Score:    p.score,
			IP:       p.ip,
			Offline:  !d.dissolve.isOnline(uid),
		})
	}
	if err := s.Push("onPlayerEnter", enter); err != nil {
		log.Error(err.Error())
		return err
	}

	p, err := playerWithSession(s)
	if err != nil {
		log.Error(err)
		return err
	}

	if err := d.playerJoin(s, true); err != nil {
		log.Error(err)
	}

	// 首局结束以后, 未点继续战斗, 此时强制退出游戏
	st := d.status()
	if st != constant.DeskStatusCreate &&
		st != constant.DeskStatusCleaned &&
		st != constant.DeskStatusInterruption {
		if err := p.syncDeskData(); err != nil {
			log.Error(err)
		}
	} else {
		d.prepare.ready(s.UID())
		d.syncDeskStatus()
		// 必须在广播消息以后调用checkStart
		d.checkStart()
	}

	// 如果已经有人申请解散
	if d.dissolve.isDissolving() {
		list := d.collectDissolveStatus()
		agreement := &protocol.DissolveResponse{
			DissolveStatus: list,
			RestTime:       d.dissolve.restTime,
		}

		uid := s.UID()
		// 如果断线前已经同意解散, 下次上线则不需要确认解散
		if d.dissolve.status[uid] {
			agreement.DissolveUid = uid
		}

		if err := s.Push("onDissolveAgreement", agreement); err != nil {
			log.Error(err)
		}

		// 向其他玩家推送最新的解散状态
		dissolveStatus := &protocol.DissolveStatusResponse{DissolveStatus: list}
		return d.group.Multicast("onDissolveStatus", dissolveStatus, func(session *session.Session) bool {
			return session.UID() != s.UID()
		})
	}

	return nil
}

func (d *Desk) doDissolve() {
	if d.status() == constant.DeskStatusDestory {
		d.logger.Debug("房间已经销毁")
		return
	}

	log.Debugf("房间: %s解散倒计时结束, 房间解散开始", d.roomNo)
	//如果不是在桌子刚创建时解散,需要进行退出处理
	if status := d.status(); status == constant.DeskStatusCreate {
		d.group.Broadcast("onDissolve", &protocol.ExitResponse{
			IsExit:   true,
			ExitType: protocol.ExitTypeDissolve,
		})
		d.destroy()
	} else {
		d.setStatus(constant.DeskStatusInterruption)
		d.roundOver()
	}
	d.logger.Debug("房间解散倒计时结束, 房间解散完成")
}

func (d *Desk) loseCoin() {
	cardCount := requireCardCount(d.opts.MaxRound)
	consume := &model.CardConsume{
		UserId:    d.creator,
		CardCount: cardCount,
		DeskId:    d.deskID,
		ClubId:    d.clubId,
		DeskNo:    d.roomNo.String(),
		ConsumeAt: time.Now().Unix(),
	}

	// 俱乐部房间
	if d.clubId > 0 {
		async.Run(func() {
			db.ClubLoseBalance(d.clubId, int64(cardCount), consume)
		})
	} else {
		p, err := d.playerWithId(d.creator)
		if err != nil {
			d.logger.Errorf("扣除玩家房卡错误，没有找到玩家，CreatorID=%d", d.creator)
			return
		}
		p.loseCoin(int64(cardCount), consume)
	}
}
