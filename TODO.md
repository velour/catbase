# TODO

## Preface

This bot was written a long time back in the spare time of a busy person. The code is often ugly and probably not idiomatic. Updating that to a modern testable codebase may be obnoxious, but every complete rewrite tends to end in stagnation, and this codebase does seem to work for the most part. You will not hurt the original author's feelings by fixing what's bad and refactoring what's good.

## Never going to get done:

* Migrate SQL to something that can marshal into structs
	* https://github.com/jmoiron/sqlx
* Fix plugin structure to not have so many exported fields. None of them need to be exporting the bot reference, for example.
* Perhaps refactor a bit so stuff can be tested
* Fix names in factoid to actually match the bucket terminology. Some things are migrated, but not everything. There should be no instances of:
	* Trigger
	* Action
	* FullText
	* Operator
* Figure out something better for time?
	* SQLite has a datetime, but the driver can't seem to handle null
	* SQLite sometimes returns a different date string, which appers to be what the driver is trying to translate from/to
* Implement factoid aliasing
* Implement an object system for the give/take commands
* Create some kind of web reference page
* Write godoc for pretty much everything and explain why functions exist
* Enter all of this into GitHub tickets
