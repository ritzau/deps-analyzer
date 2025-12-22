# Ideas for future evolution

## Binary/so-level

Start at the level of the generated artefacts. Clicking let's you see what is
inside. You may also be able to see what is accessed from other artefacts, and
what is used in external ones.

## Test coverage

This is a weird one, but the term coverage led me to think about adding quality
metrics. Test coverage is just one example. Maybe an idea?

## Integrated browser

Maybe skip the actual browser dependency and use something like electron?

## Live updates

Watch the project files for changes and update continously. At least make it
possible to trigger restarting the analysis.

## Color scheme

Make a good looking color scheme and support dark/light (auto detected by
default). Maybe dive deeper and also choose a font that we can enjoy some :)

## Investigate compiler options to also track header:header deps

Better track compile time deps to detect cycles.

## Use nm/readelf to track undefined/public symbols

Track link time dependencies.

## Caching the result

Store a cache so that we don't have to reanalyze unless there is a change.

## CI

## Test using a (headless?) browser

## Real decent logging

## External packages

- May require support of .a files

## Real so file test cases

## BUG: Only include edges to the focused target
