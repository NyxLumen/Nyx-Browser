# nyx completion powershell

Generate the autocompletion script for powershell

## Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	nyx completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
nyx completion powershell [flags]
```

## Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
  -c, --config string     config file path
  -d, --database string   database path (default "./nyx.db")
```

## See Also

* [nyx completion](nyx-completion.md)	 - Generate the autocompletion script for the specified shell

