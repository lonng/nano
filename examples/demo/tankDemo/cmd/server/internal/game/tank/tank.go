package tank

type tank struct {
	hp int
}

func newTank(hp int) *tank {
	return &tank{hp: hp}
}

func (tk tank) ReduceHp(hp int) int {
	if tk.hp > hp {
		tk.hp = tk.hp - hp
	} else {
		tk.hp = 0
	}
	return tk.hp
}
