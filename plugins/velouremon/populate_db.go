package velouremon

func (vp *VelouremonPlugin) populateDBWithBaseData() {
	vp.db.Exec(`insert into velouremon_creatures (name, defense, attack) values (?, ?, ?);`,
	"Lap Sprite", 10, 5)
	vp.db.Exec(`insert into velouremon_creatures (name, defense, attack) values (?, ?, ?);`,
	"Industry Rep", 5, 10)
	vp.db.Exec(`insert into velouremon_creatures (name, defense, attack) values (?, ?, ?);`,
	"Charpov", 10, 10)

	vp.db.Exec(`insert into velouremon_abilities (name, damage, heal, shield, weaken, critical) values (?, ?, ?, ?, ?, ?);`,
	"Heal", 0, 10, 0, 0, 0)
	vp.db.Exec(`insert into velouremon_abilities (name, damage, heal, shield, weaken, critical) values (?, ?, ?, ?, ?, ?);`,
	"Attack", 10, 0, 0, 0, 0)
	vp.db.Exec(`insert into velouremon_abilities (name, damage, heal, shield, weaken, critical) values (?, ?, ?, ?, ?, ?);`,
	"Graduate", 0, 255, 0, 0, 0)
}
