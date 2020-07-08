cssdalek
========

Exterminate unused CSS for major wins.


## Why?

Often you will find yourself wanting to use a CSS framework, like Bootstrap
or Materialize etc, only to discover that you are including large swaths of
unused CSS.

`cssdalek` helps you drop these unused CSS bits, often making the CSS you end
up serving your audience a whole lot smaller.


## Speed

There are alternatives to this tool that provide the same end result.
Possibly a better, more accurate stripping of unused rules. It's very much
possible that running your application in a browser will let you _really_ see
what rules can be dropped.

But that is slow. And so some of these other tools are also slow. So an
important goal of this tool is not to be slow. We'll have to balance speed
with accuracy.


## Accuracy

Accuracy is the flip side of speed (and memory consumption). We're aiming for
_pretty good_ ™ accuracy. We're not going to store every HTML page in memory
and run every selector like a browser, for example. But we want to drop as
much as we can to actually make this tool useful.


## FAQ

1. Descendant, child and sibling selectors are all considered the same: "an
and set". For these selectors, if all the target nodes exist anywhere, we
will include the selector. That is, the relationships are not actually
checked for.


## NOTE

This is probably not production ready, and you probably want to use
[purgecss](https://github.com/FullHuman/purgecss).


## Example

To see a quick example in action:

```sh
cssdalek -c 'example/in-*.css' -h 'example/*.html' > example/min.css
open example/index.html
```


## TODO

- [ ] Tables
- [ ] Preset whitelist
- [ ] Explicit includes
- [ ] Hook up a generic parser for random file types
- [ ] Fuzz testing
- [ ] Test various invalid HTML/CSS scenarios
