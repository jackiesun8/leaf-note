package gamedata

var (
	AccIDMin int
	AccIDMax int
)

func setGlobalConf(id int, value int) {
	switch id {
	case 1:
		AccIDMin = value
	case 2:
		AccIDMax = value
	}
}

func init() {
	type GlobalConf struct {
		ID    int
		Value int
		_     string
	}

	rf := readRf(GlobalConf{})
	for i := 0; i < rf.NumRecord(); i++ {
		r := rf.Record(i).(*GlobalConf)
		setGlobalConf(r.ID, r.Value)
	}
}
