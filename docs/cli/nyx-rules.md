# nyx rules

Rules management utility

## Synopsis

On each subcommand you'll have available at least two flags:
"username" and "id". You must either set only one of them
or none. If you set one of them, the command will apply to
an user, otherwise it will be applied to the global set or
rules.

## Options

```
  -h, --help              help for rules
  -i, --id uint           id of user to which the rules apply
  -u, --username string   username of user to which the rules apply
```

## Options inherited from parent commands

```
  -c, --config string     config file path
  -d, --database string   database path (default "./nyx.db")
```

## See Also

* [nyx](nyx.md)	 - A stylish web-based file and media browser
* [nyx rules add](nyx-rules-add.md)	 - Add a global rule or user rule
* [nyx rules ls](nyx-rules-ls.md)	 - List global rules or user specific rules
* [nyx rules rm](nyx-rules-rm.md)	 - Remove a global rule or user rule

