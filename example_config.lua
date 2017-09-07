config = {
	Channels = {
	  "#CatBaseTest"
	},
	TwitterConsumerSecret = "<Consumer Secret>",
	Reminder = {
	  MaxBatchAdd = 10
	},
	Nick = "CatBaseTest",
	LeftPad = {
	  Who = "person",
	  MaxLen = 50
	},
	Factoid = {
	  StartupFact = "speed test",
	  QuoteTime = 1,
	  QuoteChance = 0.99,
	  MinLen = 5
	},
	CommandChar = {
	  "!",
	  "¡"
	},
	FullName = "CatBase",
	Your = {
	  MaxLength = 140,
	  DuckingChance = 0.5,
	  FuckingChance = 0.15,
	  YourChance = 0.4
	},
	Emojify = {
	  Chance = 0.02
	},
	DB = {
	  File = "catbase.db",
	  Server = "127.0.0.1"
	},
	Plugins = {
	},
	Untappd = {
	  Freq = 3600,
	  Channels = {
	  },
	  Token = "<Your Token>"
	},
	LogLength = 50,
	RatePerSec = 10,
	Reaction = {
	  HarrassChance = 0.05,
	  GeneralChance = 0.01,
		NegativeHarrassmentMultiplier = 2,
	  HarrassList = {
		"msherms"
	  },
	  NegativeReactions = {
		"bullshit",
		"fake",
		"tableflip",
		"vomit"
	  },
	  PositiveReactions = {
		"+1",
		"authorized",
		"aw_yea",
		"joy"
	  }
	},
	TwitterUserKey = "<User Key>",
	MainChannel = "#CatBaseTest",
	TwitterUserSecret = "<User Secret>",
	WelcomeMsgs = {
	  "Real men use screen, %s.",
	  "Joins upset the hivemind's OCD, %s.",
	  "Joins upset the hivemind's CDO, %s.",
	  "%s, I WILL CUT YOU!"
	},
	Bad = {
	  Msgs = {
	  },
	  Hosts = {
	  },
	  Nicks = {
	  }
	},
	Irc = {
	  Server = "ircserver:6697",
	  Pass = "CatBaseTest:test"
	},
	Slack = {
	  Token = "<your slack token>"
	},
	TwitterConsumerKey = "<Consumer Key>",
	Babbler = {
	  DefaultUsers = {
		"seabass"
	  }
	},
	Type = "slack",
	Admins = {
	  "<Admin Nick>"
	},
	Stats = {
	  Sightings = {
		"user"
	  },
	  DBPath = "stats.db"
	},
	HttpAddr = "127.0.0.1:1337"
}