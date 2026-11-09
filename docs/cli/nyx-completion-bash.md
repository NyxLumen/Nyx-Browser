# nyx completion bash

Generate the autocompletion script for bash

## Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(nyx completion bash)

To load completions for every new session, execute once:

### Linux:

	nyx completion bash > /etc/bash_completion.d/nyx

### macOS:

	nyx completion bash > $(brew --prefix)/etc/bash_completion.d/nyx

You will need to start a new shell for this setup to take effect.


```
nyx completion bash
```

## Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
  -c, --config string     config file path
  -d, --database string   database path (default "./nyx.db")
```

## See Also

* [nyx completion](nyx-completion.md)	 - Generate the autocompletion script for the specified shell

