# CatBase

[![Build Status](https://travis-ci.com/velour/catbase.svg?branch=master)](https://travis-ci.com/velour/catbase)

CatBase is a bot that trolls our little corner of the IRC world and keeps our friends laughing from time to time. Sometimes he makes us angry too. He is crafted as a clone of XKCD's Bucket bot, which learns from things he's told and regurgitates his knowledge to the various channels that he lives in. I've found in many such projects that randomness can often make bots feel much more alive than they are, so CatBase is a big experiment in how great randomness is.

## Getting Help

The bot has some very basic help functions built in. Just ask for help and he'll give it to you!

> Chris: CatBase, help

> CatBase: Help topics: about variables, talker, remember, admin, factoid, beers, skeleton

> Chris: CatBase, help about

> CatBase: Hi, I'm based on godeepintir version 0.81. I'm written in Go, and you can find my source code on the internet here: http://bitbucket.org/phlyingpenguin/godeepintir

## Factoids

The primary interaction with CatBase is through factoids. These are simply just statements which can be taught to the bot and triggered at a later time. They may be triggered via the text specified in a factoid, or they may be triggered randomly by the bot when the room is silent. A simple factoid takes the shape of some trigger text, a verb, and the body of the factoid. By default, the verb is included in the full text that the bot repeats, but there are two special verbs, &lt;reply&gt; and &lt;action&gt; which do not come out in the final message.

An example:

> Chris: CatBase, Chris &lt;is&gt; amazing.

> CatBase: Okay Chris.

> Chris: CatBase, Chris

> CatBase: Chris is amazing.

When teaching facts, verbs are always enclosed with &lt;&gt;, no exceptions. Using &lt;reply&gt; causes CatBase to reply with only the body text, omitting both the verb and the trigger. Using &lt;action&gt; is similar to the &lt;reply&gt; verb except he sends an IRC action (/me) instead of a regular reply.

Factoids can be removed by telling the bot, "forget that." They can be searched and updated using the =~ operator. This works by giving a trigger, the operator, and an RE2 compatible regular expression. For example, "CatBase: Chris =~ s/amazing/the best/" would update the previous factoid to be more accurate.

### Variables

There are a few variables cooked into the bot that allow some randomness within a fact. Putting the variable in the body of a message will cause it to be replaced by an entry from the database for a given variable.

Current entries:

- $who - The person who triggered the factoid. (Random person if nobody triggered.)
- $someone - Anybody in the channel
- $digit - 0-9
- $nonzero - 1-9
- $verb, $verbs, $verbing - All common verbs in three forms
- $noun - All common nouns
- $color - Several real life colors
- $mood - Is he happy?
- $bodypart - A random body part. There are lots.
- $beer - A random beer name
- $swear - A random swear
- $me - The bot's IRC handle

## Quotes

Quotes are technically equivalent to factoids, but there is some syntactic sugar to make life easier for the members of the channel. The bot will remember what somebody said if he is told to. Later, he may trigger the fact randomly just like any other factoid and a tiny morsel of out of context quotation can appear, or the library of quotes from the user in question can be triggered manually.

An example:

> Chris: Everything I say is great.

> Jordan: CatBase, remember Chris Everything

> CatBase: Okay Jordan. Remembering what Chris said about Everything.

> Jordan: CatBase, Chris quotes

> CatBase: Chris: Everything I say is great.

## Beers

One of the fun parts about CatBase is that he remembers how many beers we have. Gamification of our drinking habits is a great motivator to get out and have more beer, so he will keep track. The keywords to know are: beers, beers++, beers = X (where X is an integer), and puke. Upon puking, your beer count is reset. Otherwise, the functionality is obvious. The spacing on "beers++" and "beers = X" is important. He also knows "bourbon++" if you're drinking something else. This counts as two beers, use sparingly!

### Untappd

One small improvement for the beer tracker is integration into [Untappd](http://untappd.com). By being friends with the [bot owner](https://untappd.com/user/phlyingpenguin), you can then tell the bot in the channel who you are and then he will automatically count beer check-ins as beers in the channel. He'll also display what the beer is and include the comment and location entered on Untappd so that your friends can be jealous and you can put in some smack talk. To register your Untappd user name, just tell the bot "reguntappd <your untappd user name>"

## Other, randomer things

The saga of bots in our channel started out with a simple bot named FredFelps which would just reply about how God hates our users at various times when they talked about the real Felps. This is still a tradition that CatBase keeps up, but could be easily modified to fit somebody else's needs.

We're also a bit OCD about the length of the IRC nicks that people use. As it turns out, if everybody agrees on a particular length, say 9 characters, then the messages and names all line up very nicely when using a fixed-width font. We like 9 characters.

Unwelcome messages also help to deter people from violating our CDO. Upon joining, users are told that they really ought to try using screen or an IRC bouncer, or just that they're terrible people. The theory is that they are terrible people.

You can use the bot to perform die rolls (character sheets may be on the way!)
by issuing a single word command in the form of XdY. "1d20" would roll a single
20-sided die, and "4d6" would roll four 6-sided dice.

## License

```
            DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
                    Version 2, December 2004

 Copyright (C) 2004 Sam Hocevar <sam@hocevar.net>

 Everyone is permitted to copy and distribute verbatim or modified
 copies of this license document, and changing it is allowed as long
 as the name is changed.

            DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
   TERMS AND CONDITIONS FOR COPYING, DISTRIBUTION AND MODIFICATION

  0. You just DO WHAT THE FUCK YOU WANT TO.
```
# c346-34515-fa22-project-rockbottom
