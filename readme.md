cssdalek
========

Exterminate unused CSS for major wins.


## Why?

Often you will find yourself wanting to use a CSS framework, like Bootstrap
or Materialize etc, only to discover that you are including large swaths of
unused CSS.

`cssdalek` helps you drop these unused CSS bits, often making the CSS you end
up serving your audience a whole lot smaller.


## How?

Usage looks like this:

```sh
cssdalek \
  --css 'example/in-*.css' \
  --html 'example/*.html' > example/min.css
```

This uses the `HTML` extractor, and will work well if you can point it to all
your markup (you can use the flags multiple times).


### Words Extractor

If you're using dynamic templates, and/or JavaScript, then you can use the
naive `Words` extractor, which assumes any word that exists can be a tag,
classname or attribute. This will get you pretty far. For example:

```sh
cssdalek \
  --css 'example/in-*.css'
  --word 'example/*.tpl'
  --word 'example/*.js' > example/min.css
```


### Includes

Finally, this tool can't recognize or detect dynamically created classnames
etc. So you can explicitly tell it about such cases either by providing
regular expressions that matches classnames or IDs, or by providing a full
selector (which can include tag, attr etc). For example:

```sh
cssdalek \
  --css 'example/in-*.css' \
  --include-class '^foo' \
  --include-id '^bar.*baz$' \
  --include-selector '#bar .foo[type=text]' \
  --word 'example/*.tpl' \
  --word 'example/*.js' > example/min.css
```

Also remember all of these can be combined. Some HTML files, some using the
word tokenizer, and others via the explicit includes.


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


## TODO

- [ ] space before !important
- [ ] Generate source mapping
- [ ] Tables
- [ ] CI tag based release pipeline
- [ ] Fuzz testing
- [ ] Test various invalid HTML/CSS scenarios
