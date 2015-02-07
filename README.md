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

Please see `examples/` in order to understand how does configuration file works. attr follows this pattern: `uid:gid:perm` where perm is written in octal as if it was typed in a shell.
