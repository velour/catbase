package velouremon

func (vp *VelouremonPlugin) populateDBWithBaseData() {
	vp.db.Exec(`insert into velouremon_creatures (name, defense, attack) values (?, ?, ?);`,
	"Lap Sprite", 10, 5)
	vp.db.Exec(`insert into velouremon_creatures (name, defense, attack) values (?, ?, ?);`,
	"Industry Rep", 5, 10)
	vp.db.Exec(`insert into velouremon_creatures (name, defense, attack) values (?, ?, ?);`,
	"Charpov", 10, 10)

	vp.db.Exec(`insert into velouremon_abilities (name, damage, heal, shield, weaken, critical) values (?, ?, ?, ?, ?, ?);`,
	"Procrastinate", 0, 0, 10, 0, 0)
	vp.db.Exec(`insert into velouremon_abilities (name, damage, heal, shield, weaken, critical) values (?, ?, ?, ?, ?, ?);`,
	"Defend", 0, 0, 5, 0, 0)
	vp.db.Exec(`insert into velouremon_abilities (name, damage, heal, shield, weaken, critical) values (?, ?, ?, ?, ?, ?);`,
	"Graduate", 0, 255, 0, 0, 0)
}
