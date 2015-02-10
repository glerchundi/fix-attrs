# fix-attrs
modifies file attributes (ownership &amp; permission) based on a configuration file

Provide all required arguments:
```
fix-attrs fix --format yaml file.cfg
```

It detects format based on file extension:
```
fix-attrs fix file.yml
```

Compile compatible versions:
```
OS=(linux darwin)
for i in "${OS[@]}"; do
    CGO_ENABLED=0 GOOS=$i GOARCH=amd64 \
    go build                           \
      -o fix-attrs-0.2.0-$i-amd64      \
      -a                               \
      -tags netgo                      \
      -ldflags '-w'                    \
      .
done
```

Please see `examples/` in order to understand how does configuration file works. attr follows this pattern: `uid:gid:perm` where perm is written in octal as if it was typed in a shell.

`uid` and `gid` are resolved following this rules: [http://www.gnu.org/software/coreutils/manual/html_node/Disambiguating-names-and-IDs.html](Disambiguating user names and IDs)