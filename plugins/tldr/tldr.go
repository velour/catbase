package tldr

import (
	"fmt"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"

	"github.com/rs/zerolog/log"

	"github.com/james-bowman/nlp"
)

var (
	THESE_ARE_NOT_THE_WORDS_YOU_ARE_LOOKING_FOR = []string{"p", "z", "i", "c", "e", "s", "x", "n", "b", "t", "d", "m", "r", "a", "f", "l", "w", "o", "g", "h", "v", "k", "y", "j", "u", "q", "th", "wu", "qt", "so", "ru", "pm", "in", "is", "am", "me", "on", "by", "kw", "hu", "bg", "ob", "re", "wx", "go", "hl", "vc", "bl", "rg", "wr", "cw", "pj", "tf", "nr", "aw", "qc", "it", "cj", "or", "ty", "hk", "be", "wc", "de", "lf", "mj", "bw", "at", "as", "gd", "ww", "ko", "og", "gg", "cz", "an", "mh", "we", "rb", "mv", "uk", "wt", "us", "hq", "if", "mu", "pn", "js", "my", "ol", "ul", "io", "lm", "do", "cd", "fo", "no", "vg", "lu", "dg", "zu", "sv", "wn", "fu", "dk", "tv", "la", "sn", "wb", "pc", "he", "pk", "ii", "wm", "up", "bo", "ca", "fd", "uh", "hh", "al", "id", "bd", "uw", "co", "pf", "ez", "df", "ro", "et", "dh", "ui", "gl", "st", "rl", "ev", "jj", "fp", "hc", "en", "eh", "rp", "ka", "rj", "bm", "oh", "tb", "ix", "ad", "cg", "ny", "rn", "cn", "dc", "vp", "jm", "tp", "om", "ok", "ms", "wp", "hi", "aj", "oc", "sq", "hp", "yu", "sk", "dx", "eg", "ip", "bk", "hz", "pa", "fg", "rh", "tx", "ve", "za", "ht", "ie", "el", "ma", "xi", "ou", "dp", "nu", "mw", "mf", "md", "fl", "mb", "mr", "ld", "uc", "il", "ln", "mm", "ur", "ed", "pd", "le", "jc", "az", "un", "mi", "dm", "wy", "jd", "oe", "to", "pb", "dr", "kb", "pp", "na", "rx", "os", "nb", "yn", "ci", "gc", "ex", "dt", "au", "fi", "np", "nc", "po", "va", "rd", "sc", "ws", "cu", "se", "di", "km", "ga", "ac", "ft", "lc", "fa", "im", "vs", "ar", "mo", "sa", "sg", "uv", "xp", "je", "eq", "lt", "eu", "cc", "wa", "dj", "ls", "cm", "wi", "dl", "ct", "fx", "yo", "da", "vb", "of", "nj", "hr", "em", "iv", "nn", "rw", "fs", "ye", "um", "ni", "ne", "du", "oo", "bp", "gs", "fw", "nt", "es", "fc", "ti", "cb", "cv", "gb", "bc", "pr", "fr", "aa", "mt", "ir", "gp", "oz", "mg", "tc", "hb", "sl", "af", "bt", "ch", "sd", "jp", "lb", "rs", "ep", "ef", "rr", "fy", "tu", "dv", "xl", "ss", "tt", "ap", "nm", "mn", "nd", "pe", "op", "ng", "tn", "ge", "ts", "gr", "ce", "mx", "ab", "ic", "yr", "ot", "ai", "pi", "rv", "hs", "ae", "tm", "sp", "sh", "gt", "nh", "ho", "cl", "ll", "fm", "gi", "ta", "db", "ph", "ia", "pt", "bi", "ha", "ds", "ea", "lg", "bs", "ja", "ns", "wv", "nw", "sm", "ff", "ah", "sb", "td", "fe", "ak", "rf", "ps", "ky", "pl", "br", "lo", "ml", "dd", "cp", "cs", "rt", "ri", "gm", "sf", "kg", "ut", "si", "mc", "vt", "lp", "cf", "rm", "ag", "vi", "ec", "ba", "rc", "cr", "pg", "ee", "ra", "ks", "sw", "av", "te", "hd", "nz", "bb", "er", "jr", "tr", "nv", "ya", "nl", "li", "su", "mp", "sr", "ted", "bid", "can", "the", "nat", "car", "wan", "dig", "neo", "enb", "pvc", "dod", "fri", "dvd", "cia", "tex", "wed", "une", "how", "inn", "lid", "mia", "ltd", "los", "are", "yen", "cho", "dui", "inc", "win", "col", "upc", "bed", "dsc", "ste", "aye", "nhs", "dow", "tue", "cio", "ooo", "cas", "thu", "sea", "cut", "mpg", "rrp", "tel", "its", "ips", "pts", "own", "kit", "mug", "has", "sku", "nbc", "dip", "acm", "boy", "end", "ids", "him", "est", "son", "ict", "mac", "iii", "gmt", "max", "per", "xml", "big", "bin", "law", "sap", "ala", "art", "cir", "lip", "bat", "top", "eco", "sol", "van", "had", "buf", "rip", "ads", "usa", "wma", "seq", "pop", "int", "rid", "rna", "sim", "abs", "hit", "but", "wal", "ati", "doe", "eye", "geo", "old", "arg", "usb", "uni", "php", "etc", "diy", "leo", "tgp", "mud", "msn", "fee", "rpg", "las", "ide", "sic", "min", "aid", "avi", "ons", "non", "mel", "div", "ppc", "day", "fat", "saw", "cet", "cow", "mls", "pst", "why", "phi", "bra", "mae", "tom", "fin", "sub", "irc", "gpl", "led", "fan", "low", "ten", "gif", "ate", "man", "cat", "die", "ton", "tmp", "rec", "two", "ddr", "our", "gsm", "pet", "guy", "dev", "cup", "vol", "one", "you", "mag", "dee", "pit", "mba", "lee", "job", "boc", "pmc", "cfr", "bee", "vii", "llp", "too", "tap", "for", "bob", "fit", "men", "met", "mem", "por", "www", "cgi", "soa", "jvc", "tft", "ccd", "liz", "ice", "dat", "ali", "box", "llc", "sec", "bus", "ash", "bag", "gay", "all", "tub", "sox", "ibm", "sas", "gig", "qui", "pty", "dns", "air", "nor", "bug", "mid", "pas", "icq", "sys", "das", "lil", "cnn", "jim", "buy", "yes", "dam", "del", "hot", "qld", "new", "say", "ist", "joe", "may", "cab", "cds", "nav", "ind", "pct", "pos", "dec", "pod", "vic", "psi", "san", "cms", "gem", "tax", "uri", "got", "atm", "vcr", "lab", "cvs", "hon", "let", "bow", "des", "cbs", "eos", "lcd", "inf", "ave", "act", "red", "pie", "apt", "her", "alt", "ant", "key", "ppm", "tan", "few", "sip", "out", "kde", "pic", "gym", "age", "mat", "add", "use", "asn", "pgp", "lou", "jan", "oct", "pay", "tag", "mix", "any", "vhs", "fix", "pal", "tri", "thy", "war", "nov", "ray", "leu", "fda", "see", "vid", "std", "gmc", "dry", "spa", "aaa", "con", "ups", "fax", "yet", "gel", "sao", "lap", "sun", "rss", "nhl", "gen", "mtv", "mil", "cod", "not", "run", "net", "msg", "eau", "plc", "was", "var", "dos", "put", "rat", "his", "won", "oem", "tin", "doc", "try", "mom", "rap", "mlb", "row", "she", "flu", "opt", "usd", "abu", "ssl", "ana", "jpg", "eat", "cdt", "ins", "aim", "isp", "seo", "les", "bye", "ann", "tip", "rfc", "log", "ski", "irs", "faq", "dan", "chi", "nut", "wax", "fly", "dts", "fun", "gbp", "sen", "hey", "sue", "bbc", "ace", "tea", "avg", "sky", "feb", "rom", "eng", "toy", "sep", "src", "hip", "hub", "ghz", "eds", "lot", "val", "dot", "hiv", "pda", "dir", "ask", "dsl", "zum", "dna", "tcp", "cad", "fcc", "tee", "aka", "tim", "sql", "zoo", "don", "due", "mai", "cry", "vpn", "who", "dim", "mar", "cop", "gps", "erp", "acc", "pro", "cap", "ink", "phd", "pam", "url", "aug", "pin", "raw", "gnu", "amy", "ben", "ext", "web", "aol", "ago", "pac", "odd", "ent", "hat", "zus", "lib", "ban", "cos", "utc", "der", "fed", "apr", "ion", "roy", "cam", "app", "wet", "ram", "nil", "fox", "mrs", "arc", "arm", "via", "jar", "obj", "dom", "kai", "rio", "jam", "nyc", "len", "pub", "bad", "mas", "set", "hop", "bon", "gst", "gun", "ata", "rca", "ira", "eva", "rev", "sur", "sie", "lat", "sam", "pdt", "mhz", "egg", "tvs", "pee", "rpm", "img", "ref", "pot", "far", "kid", "map", "pan", "tba", "cal", "now", "and", "sad", "jul", "psp", "fbi", "jun", "hrs", "ham", "und", "rod", "wav", "dem", "way", "pad", "nfl", "eve", "rug", "soc", "amd", "usc", "mic", "tar", "fur", "yea", "iso", "sum", "vip", "amp", "str", "oak", "vat", "fog", "duo", "sig", "get", "sir", "crm", "kim", "lie", "gba", "oil", "spy", "bit", "aud", "foo", "den", "yrs", "pix", "res", "sit", "wow", "isa", "ada", "una", "que", "lit", "pig", "fig", "gdp", "bbs", "nec", "nam", "sms", "tab", "bay", "css", "gtk", "lan", "urw", "qty", "hwy", "aus", "fwd", "bio", "api", "toe", "sri", "pcs", "bar", "mit", "von", "dog", "rep", "ser", "wit", "ceo", "sci", "edt", "cst", "sin", "bmw", "hay", "eur", "kay", "pdf", "mod", "dis", "zen", "ian", "ing", "rim", "tie", "pci", "ear", "nsw", "ftp", "med", "reg", "wto", "ver", "gui", "leg", "pat", "off", "dad", "abc", "org", "usr", "jay", "gap", "ron", "til", "mon", "com", "biz", "rob", "era", "gcc", "asp", "did", "epa", "jet", "par", "nba", "loc", "gas", "mad", "six", "gis", "def", "ken", "pre", "exp", "bet", "pen", "mph", "dpi", "joy", "cpu", "ran", "lol", "sat", "jon", "lay", "lbs", "zip", "ill", "rows", "pipe", "seal", "deck", "sand", "thin", "shoe", "sick", "dose", "till", "cafe", "lets", "andy", "semi", "cats", "cake", "gang", "greg", "dial", "luck", "belt", "tube", "rail", "folk", "tiny", "okay", "hist", "lift", "lisa", "mall", "wing", "neck", "fell", "yard", "busy", "tone", "sean", "pour", "gate", "tion", "dust", "wiki", "kent", "adds", "bugs", "bone", "bang", "alex", "ward", "meat", "roof", "kiss", "peer", "seed", "para", "cute", "rush", "mpeg", "yoga", "lamp", "rico", "phil", "pmid", "http", "bulk", "glad", "wins", "rack", "aged", "scan", "bold", "boss", "ross", "anna", "solo", "tall", "grey", "pdas", "beds", "ryan", "nova", "exam", "anne", "pump", "wake", "plot", "nick", "nasa", "drum", "pull", "foto", "ease", "tabs", "voip", "grid", "pine", "tend", "gulf", "echo", "rick", "char", "hunt", "thai", "fred", "chip", "mill", "suit", "bits", "dont", "burn", "labs", "twin", "earn", "jane", "jose", "beer", "dear", "alan", "misc", "push", "sole", "boot", "laid", "clay", "weak", "milk", "blvd", "arab", "wise", "rome", "odds", "vary", "gary", "marc", "sons", "leaf", "loop", "rice", "hate", "demo", "cuba", "gray", "silk", "kate", "slot", "adam", "wolf", "dish", "fits", "kick", "meal", "navy", "hurt", "tank", "bowl", "slip", "mens", "cuts", "mile", "mars", "lock", "node", "rear", "caps", "pill", "legs", "meta", "mint", "crew", "spin", "babe", "wash", "warm", "draw", "aims", "lens", "ieee", "pure", "corp", "visa", "jean", "soap", "bond", "unix", "poll", "axis", "guns", "dean", "mesh", "hero", "acts", "punk", "holy", "duke", "wave", "pace", "wage", "keys", "iran", "dawn", "carl", "coat", "exit", "rica", "matt", "soil", "kits", "tony", "doll", "seek", "peru", "nike", "lose", "reed", "mice", "bike", "temp", "perl", "vast", "cook", "plug", "wrap", "mood", "quiz", "ages", "kill", "lane", "beam", "tops", "jeff", "bell", "shut", "salt", "ncaa", "thou", "peak", "mask", "euro", "evil", "coal", "yeah", "runs", "pair", "ride", "pets", "lion", "goto", "hole", "neil", "beef", "bass", "hats", "diff", "surf", "onto", "rain", "hook", "cord", "grow", "crop", "spot", "eric", "lite", "nine", "faqs", "slow", "hide", "utah", "arms", "sing", "tons", "beat", "kept", "hang", "wars", "fear", "hood", "moon", "dogs", "math", "fame", "whom", "mine", "cape", "toll", "bids", "seat", "eggs", "dell", "fans", "lady", "ruby", "mins", "bird", "stem", "rise", "drew", "dual", "bars", "rare", "tune", "corn", "wear", "puts", "grew", "bags", "trek", "jazz", "fail", "ties", "beta", "brad", "jury", "font", "tail", "lawn", "soup", "byte", "nose", "oclc", "bath", "juan", "roll", "zero", "thru", "jews", "trim", "null", "cent", "acid", "espn", "spam", "quit", "lung", "tape", "wire", "clip", "todd", "blow", "doug", "sees", "zoom", "knew", "bull", "cole", "mart", "tale", "lynn", "iowa", "lack", "docs", "gain", "bear", "coin", "fake", "duty", "cure", "arch", "vice", "hdtv", "asin", "bomb", "harm", "hong", "deer", "dave", "desk", "disk", "void", "iron", "atom", "flag", "oven", "aids", "noon", "soul", "felt", "cast", "cams", "joel", "ends", "proc", "icon", "boat", "mate", "disc", "chef", "isle", "slim", "luke", "comp", "gene", "fort", "gone", "fill", "pete", "spec", "camp", "penn", "midi", "tied", "snow", "dale", "oils", "sept", "unto", "inch", "died", "kong", "pays", "rank", "lang", "stud", "fold", "ones", "gave", "hire", "seem", "ipod", "phys", "pole", "mega", "bend", "moms", "glen", "rich", "drop", "guys", "tags", "lips", "pond", "load", "pick", "rose", "wait", "walk", "tire", "chad", "fuel", "josh", "drag", "soft", "ripe", "rely", "scsi", "task", "miss", "wild", "heat", "nuts", "nail", "span", "mass", "joke", "univ", "foot", "pads", "inns", "cups", "cold", "shot", "pink", "foam", "root", "edge", "poem", "ford", "oral", "asks", "bean", "bias", "xbox", "pain", "palm", "wind", "sold", "swim", "nano", "goal", "ball", "dvds", "loud", "rats", "jump", "stat", "cruz", "bios", "firm", "thee", "lots", "ruth", "pray", "pope", "jeep", "bare", "hung", "gear", "army", "mono", "tile", "diet", "apps", "skip", "laws", "path", "flow", "ciao", "knee", "prep", "flat", "chem", "jack", "zone", "hits", "pros", "cant", "wife", "goes", "hear", "lord", "farm", "sara", "eyes", "joan", "duck", "poor", "trip", "mike", "dive", "dead", "fiji", "audi", "raid", "gets", "volt", "ohio", "dirt", "fair", "acer", "dist", "isbn", "geek", "sink", "grip", "host", "watt", "pins", "reno", "dark", "polo", "rent", "horn", "wood", "prot", "frog", "logs", "sets", "core", "debt", "snap", "race", "born", "pack", "fish", "jpeg", "mini", "pool", "swap", "rest", "flip", "deep", "boys", "buzz", "nuke", "iraq", "boom", "calm", "fork", "troy", "ring", "mary", "prev", "zope", "gmbh", "skin", "fees", "sims", "tray", "pass", "sage", "java", "uses", "asia", "cool", "suse", "door", "cave", "wool", "feet", "told", "rule", "ways", "eyed", "vote", "grab", "oops", "wine", "wall", "thus", "tree", "trap", "fool", "hair", "karl", "dies", "paid", "ship", "anti", "hall", "jail", "feed", "safe", "ipaq", "hold", "comm", "deal", "maps", "lace", "hill", "ugly", "hart", "ment", "tool", "idea", "fall", "biol", "late", "lies", "cnet", "song", "took", "treo", "gods", "male", "fund", "mode", "poly", "ears", "went", "lead", "fist", "band", "mere", "cons", "sent", "taxi", "nice", "logo", "move", "kind", "huge", "bush", "hour", "worn", "shaw", "fine", "expo", "came", "deny", "bali", "judy", "trio", "cube", "rugs", "fate", "role", "gras", "wish", "hope", "menu", "tour", "lost", "mind", "oval", "held", "soma", "soon", "href", "benz", "wifi", "tier", "stop", "earl", "port", "seen", "guam", "cite", "cash", "pics", "drug", "copy", "mess", "king", "mean", "turn", "stay", "rope", "dump", "near", "base", "face", "loss", "hose", "html", "chat", "fire", "sony", "pubs", "lake", "paul", "mild", "none", "step", "half", "sort", "clan", "sync", "mesa", "wide", "loan", "hull", "golf", "toys", "shed", "memo", "girl", "tide", "funk", "town", "reel", "risk", "bind", "rand", "buck", "bank", "feel", "meet", "usgs", "acre", "lows", "aqua", "chen", "emma", "tech", "pest", "unit", "fact", "fast", "reef", "edit", "auto", "plus", "chan", "tips", "beth", "rock", "mark", "else", "jill", "sofa", "true", "dans", "viii", "kids", "talk", "tent", "dept", "hack", "dare", "hawk", "lamb", "bill", "word", "ever", "done", "land", "says", "upon", "five", "past", "arts", "gold", "able", "junk", "tell", "lucy", "hans", "poet", "epic", "cars", "sake", "sans", "sure", "lean", "once", "away", "self", "dude", "luis", "cell", "head", "film", "alto", "term", "baby", "keep", "gore", "cult", "dash", "cage", "divx", "hugh", "hand", "jake", "eval", "ping", "flux", "star", "muze", "oman", "easy", "blue", "rage", "adsl", "four", "prix", "hard", "gift", "avon", "rays", "road", "walt", "acne", "libs", "undo", "club", "east", "dana", "halo", "body", "sell", "gays", "give", "exec", "side", "park", "blog", "less", "play", "maui", "cart", "come", "test", "vids", "yale", "july", "cost", "plan", "june", "doom", "owen", "bite", "issn", "myth", "live", "note", "week", "weed", "oecd", "dice", "quad", "dock", "mods", "hint", "msie", "team", "left", "look", "west", "buys", "pork", "join", "barn", "room", "teen", "fare", "sale", "asus", "food", "bald", "fuji", "leon", "mold", "dame", "jobs", "herb", "card", "alot", "york", "idle", "save", "call", "main", "john", "love", "form", "rate", "text", "cove", "casa", "shop", "eden", "incl", "size", "down", "care", "game", "reid", "both", "flex", "rosa", "hash", "lazy", "same", "case", "carb", "open", "link", "file", "sign", "much", "even", "pens", "show", "code", "worm", "long", "deaf", "mats", "want", "blah", "mime", "feof", "usda", "keen", "peas", "urls", "area", "take", "type", "send", "owns", "line", "made", "must", "ebay", "item", "zinc", "guru", "real", "levy", "grad", "bras", "part", "days", "kyle", "know", "life", "pale", "gaps", "tear", "full", "mail", "does", "nest", "said", "nato", "user", "gale", "many", "stan", "idol", "need", "read", "book", "very", "moss", "each", "cork", "high", "mali", "info", "dome", "well", "heel", "yang", "good", "then", "best", "such", "city", "post", "them", "make", "data", "dumb", "most", "last", "work", "used", "next", "into", "year", "feat", "ntsc", "over", "usps", "just", "name", "conf", "glow", "list", "back", "oaks", "erik", "date", "find", "than", "paso", "norm", "like", "some", "ware", "were", "been", "jade", "foul", "keno", "view", "seas", "help", "pose", "mrna", "goat", "also", "here", "sail", "when", "only", "sega", "cdna", "news", "what", "bolt", "gage", "site", "they", "time", "urge", "smtp", "kurt", "neon", "ours", "lone", "cope", "free", "lime", "kirk", "bool", "page", "home", "will", "spas", "more", "jets", "have", "intl", "your", "yarn", "knit", "from", "with", "this", "pike", "that", "hugo", "gzip", "ctrl", "bent", "laos"}
)

type TLDRPlugin struct {
	Bot     bot.Bot
	History []string
	Users []string
	Index   int
}

func New(b bot.Bot) *TLDRPlugin {
	plugin := &TLDRPlugin{
		Bot:     b,
		History: []string{},
		Users: []string{},
		Index:   0,
	}
	b.Register(plugin, bot.Message, plugin.message)
	b.Register(plugin, bot.Help, plugin.help)
	return plugin
}

func (p *TLDRPlugin) message(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	lowercaseMessage := strings.ToLower(message.Body)
	if lowercaseMessage == "tl;dr" {
		for _, str := range p.History {
			fmt.Println(str)
		}

		nTopics := p.Bot.Config().GetInt("TLDR.Topics", 5)

		vectoriser := nlp.NewCountVectoriser(THESE_ARE_NOT_THE_WORDS_YOU_ARE_LOOKING_FOR...)
		lda := nlp.NewLatentDirichletAllocation(nTopics)
		pipeline := nlp.NewPipeline(vectoriser, lda)
		docsOverTopics, err := pipeline.FitTransform(p.History...)

		if err != nil {
			log.Error().Err(err)
			return false
		}

		bestScores := make([]float64, nTopics)
		bestDocs := make([]string, nTopics)
		bestUsers := make([]string, nTopics)

		dr, dc := docsOverTopics.Dims()
		for doc := 0; doc < dc; doc++ {
			for topic := 0; topic < dr; topic++ {
				score := docsOverTopics.At(topic, doc)
				if score > bestScores[topic] {
					bestScores[topic] = score
					bestDocs[topic] = p.History[doc]
					bestUsers[topic] = p.Users[doc]
				}
			}
		}

		topicsOverWords := lda.Components()
		tr, tc := topicsOverWords.Dims()

		vocab := make([]string, len(vectoriser.Vocabulary))
		for k, v := range vectoriser.Vocabulary {
			vocab[v] = k
		}

		response := "Here you go captain 'too good to read backlog':\n"

		for topic := 0; topic < tr; topic++ {
			bestScore := -1.
			bestTopic := ""
			for word := 0; word < tc; word++ {
				score := topicsOverWords.At(topic, word)
				if score > bestScore {
					bestScore = score
					bestTopic = vocab[word]
				}
			}
			response += fmt.Sprintf("Topic #%d : %s\n", topic, bestTopic)
			response += fmt.Sprintf("\t<%s>%s\n", bestUsers[topic], bestDocs[topic])
		}

		p.Bot.Send(bot.Message, message.Channel, response)

		return true
	}

	if shouldKeepMessage(lowercaseMessage) {
		currentHistorySize := len(p.History)
		maxHistorySize := p.Bot.Config().GetInt("TLDR.HistorySize", 1000)
		if currentHistorySize < maxHistorySize {
			p.History = append(p.History, lowercaseMessage)
			p.Users = append(p.Users, message.User.Name)
			p.Index = 0
		} else {
			if currentHistorySize > maxHistorySize {
				// We could resize this but we want to prune the oldest stuff, and
				// I don't care to do this correctly so might as well not do it at all
			}

			if p.Index >= currentHistorySize {
				p.Index = 0
			}

			p.History[p.Index] = lowercaseMessage
			p.Users[p.Index] = message.User.Name
			p.Index++
		}
	}
	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *TLDRPlugin) help(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.Bot.Send(bot.Message, message.Channel, "tl;dr")
	return true
}

func shouldKeepMessage(message string) bool {
	return true
}
