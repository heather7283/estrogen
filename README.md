# estrogen

This tool copies files from source to destination according to filter rules,
optionally converting them with external commands and renaming. Intended to
be used for media **trans**coding.

It's easier to understand with an example:

```toml
src = '~/music/lossless/'
dst = '~/music/converted/'

[[filter]]
exclude = '^unsorted/$'

[[rename]]
pattern = '[\?\*:"]'
replacement = '_'

[[rule]]
src = '^(.*)\.flac$'
dst = '${1}.opus'
cmd = 'ffmpeg -y -loglevel error -i "$1" "$2"'
```

Running estrogen with the config shown above will recreate file hierarchy from
`~/music/lossless` in `~/music/converted`, skipping `unsorted` subdirectory,
replacing a set of "special" characters with underscores, and converting flac
files to opus using ffmpeg.

Similarly to make, estrogen will not waste time processing files that have not
changed in the source since they were copied to dest.

See [estrogen.toml](estrogen.toml) for more thorough explanation.

