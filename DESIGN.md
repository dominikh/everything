# Target audience
_Everything_ will be designed for nerds/power-users who have to manage a large, self-hosted amount of data on Linux desktops.

## Nerds
We will not attempt to bring the semantic desktop to mainstream users.
Such a task would be too all-encompassing,
requiring cooperation from every application,
restrictions to the powerfulness of _Everything_,
and generally a much deeper revolution of the desktop metaphor.
Instead of taking on all of this,
we'll focus on people who are already comfortable with hacking on things,
using the shell, and tweaking things as they see fit.

## Large amounts of data
_Everything_ will be able to handle large amounts of data, meaning tens of terabytes and millions of files.
It should be the ideal tool for a data hoarder to manage their hoard.

## Self-hosted
_Everything_ will most likely assume locally accessible data.
While this doesn't exclude NASs and other data storage in the local network, it will exclude cloud-hosted data.
While it probably would be possible to write adapters that interface with cloud storage,
we won't do so in this project, and we will assume that data can be read fast and often.
Data that exists in the cloud should also be mirrored to locally accessible storage that one owns, anyway.

## Linux
We will target Linux and may make use of Linux-specific features.
Note that we really do mean Linux, not Unix-likes.

## Desktops
The end-user will manage and use _Everything_ from their desktop system.
There will not be a centrally managed instance that is shared by many users, nor will there be browser-based UIs.

# Data storage
TODO.

- Why not plain tags?
- Why graphs instead of relational database?
- Why triples instead of property graphs?
- Why RDF?

# UI
## Interactivity in indexing
Indexing gets complicated by ambiguities.
Movie files without existing metadata get tagged based on their file names, which might not have unique results.
A file being overwritten may mean user-applied tags are no longer valid, but it might also not.
Instead of trying to solve all of these problems automatically, accept that sometimes we'll need user input.
Provide a list of open questions and a way to decide on them.
