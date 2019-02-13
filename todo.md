Ironic isn't it? It could organize others, but not itself (yet).

(I usually have these files ignored by git, but I am adding this one since I
want to eliminate it or codify it into something proper).

Hopefully I can incorporate this file I use in other projects already with this
one, though it may take some work. I'm liking that idea.

But anyways here are some musing about what I think this should do:

TODO: Move to README.

---

# General Idea
I want this project to codify the various things I have running around in my
head that I need to get done.

# Requirements
## Have a way to remind me of scheduled items that may or may not be regular
Items that are regularly occurring should have some way to make them happen e.g.
daily, weekly, monthly, yearly, etc.

The time to "remind me" should be configurable per action so that it happens:
* At the time/day of the event
* `x` hours/days/minutes/... before the event
* Randomly during _free time_
* Never

The method for which it reminds me is up in the air. Email, Android™
notification, SMS, etc.

## Remind me of itches I need to scratch
Examples:
* This thing
* Moving from Spotify™ to something less ™

## Loose Organizational Structure
This is the part I'm least sure about.

Currently on my Android™ todo app I use a tagging system with these tags:
* Blog
  - A bunch of targets I want to blog about at some point. Generally quick high
    level topic sentences. This is because the app sucks though, it would be
    nice to have documents there. Though I don't think anything more complicated
    than markdown is necessary.
  - I basically never use this
* Interviewing
  - Dead
  - Though won't always be dead so we might want to do something for "dead"
    tags? Or maybe not even have that be possible to do. A file that had all of
    its links deleted no longer exists right, why should the tag exist separate
    of the tasks that it describes?
* Maintenance
  - Stuff I use for "crap" I need to do. Usually something I don't actually want
    to do. I don't know if that's worth encoding, but I thought it was at one point.
  - By encoding this as "crap" it's a nice catchall for stuff I don't care
    enough to categorize while I can also prioritize this specialize since it's
    sort of a "shortcut" in my system. The reason it's a shortcut is because I
    _want_ to avoid this ambiguity on things: tasks should have a reason and a
    deadline, and either be one off or scheduled regularly. I should also want
    to do them, so the more things I shove in this I need to figure out how to
    stop doing those tasks (or make them more bearable).
* School
  - School work. This will go away soon so don't bother too much about this.
* Way Cooler
  - This is very specific, which is good. Though this does potentially beg a
    _hierarchy_ that I might want to encode. Will I have a "code" or "projects"
    where Way Cooler and wlroots-rs both exist under? Is there a need for this
    sort of hierarchy? It sounds more awkward to work with so lets avoid it since
    it's easy to add later potentially but I don't know how much it adds.
  - Though "Way Cooler" is the correct styleization, I should force
    non-whitespace in my tags since that's annoying and forces them to be short
    and descriptive. We should revisit this.

So I might want a tagging system.

## Issue Tracking Linking/Creation
It would be nice that I could link to e.g. a bug tracker in my tasks or be able
to export a task into a bug tracker. This is not a replacement for making my
tasks public mind you, that is a different problem. This is about interacting
with 3rd party systems.

This, of course, rubs up a bigger problem however: where do we draw the line
with 3rd party programs? Sharing on Facebook™ is too much, but wouldn't it be
nice to make wormy remind me about my tasks?

We'll punt on this problem for now, but something to consider.

## Syncing
For now I'm not going to worry about this because this is a totally separate
problem from what I want to mainly fix. However at some point I want to be able
to sync this system across multiple devices. So I'll need to figure out how to
do that.

I don't want conflicts, so no git or dropbox style management. Working offline
might be optional, it depends on how often that happens. For now it's fine to
work with the assumption of 100% internet access. I'm not opposed to
centralization (e.g on a web server).


# Encoding Time
Generally I need a "busy" time and a "non-busy" time. These might need to be
subdivided, but to keep it simple that should be good enough for now.

For non-busy time it should be possible to query for things I can do.

# Content Representation
Forcing task names is stupid, that's not always something I want. However not
forcing tags is stupid too, I want structure. 

Priority shouldn't be a "color" based system where there's e.g lowest to
highest. It's impossible to compare those things. Severity should be based on:
* Time
  - Earlier things are more important
* Tags
  - e.g.: things in `work` should be dealt with faster than `way-cooler` for
    example. But there might be variability on the individual task so we might
    need an override or a better way to encode this.


Should use markdown (just like this file) using the standard syntax + code
blocks like so: 

```go
func main() {
	fmt.Println("Hello World")
}
```

What Github™ renders is good basically.


## Rational for Markdown
* Easy to write and pleasant to read (rendered).
* (Relatively) standard.
* Is a non-binary format for easy data processing.
