# Essh

**Now Essh is on unstable stage. API and code may be broken in future. And document lacks. sorry!**

Essh is an extended ssh client command. The features are the following:

* **Simple**: A single binary CLI tool. Essh simply wraps `ssh` command. You can use it in the same way as `ssh`.
* **Configuration as code**: You can write SSH client configuration in Lua.
* **Hooks**: Essh supports hooks that execute commands when it connects a remote server.
* **Servers List Management**: Essh provides utility commands for managing hosts, that list and classify servers by using tags.
* **Zsh Completion**: Essh provides built-in zsh completion code.
* **Per-Project Configuration**: Essh supports per-project configuration. This allows you to change SSH hosts config by changing current working directory.
* **Tasks**: Task is code that runs on remote and local servers. You can use it to automate your system administration tasks.
* **Modules**: Essh provides modular system that allows you to use, create and share reusable Lua code easily.

Table of contents

* [Getting Started](#getting-started)
  * [Installation](#installation)
  * [Connecting via SSH](#connecting-via-ssh)
  * [Zsh Completion](#zsh-completion)
* [Configuration](#configuration)
  * [Syntax](#syntax)
  * [Configuration files](#configuration-files)
  * [Hosts](#hosts)
  * [Tasks](#tasks)
  * [Modules](#modules)
  * [Libraries](#libraries)
  * [Drivers](#drivers)
* [Command line options](#command-line-options)
* [Integrating other SSH related commands](#integrating-other-ssh-related-commands)
* [Author](#author)
* [License](#license)

## Getting Started

### Installation

Essh is provided as a single binary. You can download it and drop it in your $PATH.

[Download latest version](https://github.com/kohkimakimoto/essh/releases/latest)

After installing Essh, run the `essh --version` in your terminal to check working.

```
$ essh --version
0.26.0 (9e0768e54c2131525e0e7cfb8d666265275861bc)
```

### Connecting via SSH

To get started, create and edit `~/.essh/config.lua`. This is a main configuration file for Essh. The configuration is written in [Lua](https://www.lua.org/) programming language.

```lua
host "web01.localhost" {
    ForwardAgent = "yes",
    HostName = "192.168.0.11",
    Port = "22",
    User = "kohkimakimoto",
}

host "web02.localhost" {
    ForwardAgent = "yes",
    HostName = "192.168.0.12",
    Port = "22",
    User = "kohkimakimoto",
}
```

This configuration automatically generates the below ssh config to the temporary file like the `/tmp/essh.ssh_config.260398422` whenever you run `essh`.

```
Host web01.localhost
    ForwardAgent yes
    HostName 192.168.0.11
    Port 22
    User kohkimakimoto

Host web02.localhost
    ForwardAgent yes
    HostName 192.168.0.12
    Port 22
    User kohkimakimoto
```

Essh uses this generated config file by default. If you run the below command

```
$ essh web01.localhost
```

Essh internally runs the `ssh` command like the following.

```
ssh -F /tmp/essh.ssh_config.260398422 web01.localhost
```

Therefore you can connect with a ssh server using Lua config.

Essh also automatically removes the temporary file when the process finishes. So you don't have to be conscious of the real ssh configuration in the normal operations.

### Zsh Completion

Essh supports zsh completion that lists SSH hosts. If you want to use it, add the following code in your `~/.zshrc`

```
eval "$(essh --zsh-completion)"
```

And then, edit your `~/.essh/config.lua`. Try to add the `description` property as the following.

```lua
host "web01.localhost" {
    ForwardAgent = "yes",
    HostName = "192.168.0.11",
    Port = "22",
    User = "kohkimakimoto",
    -- add description
    description = "web01 development server",
}

host "web02.localhost" {
    ForwardAgent = "yes",
    HostName = "192.168.0.12",
    Port = "22",
    User = "kohkimakimoto",
    -- add description
    description = "web02 development server",
}
```

You will get completion about hosts.

```
$ essh [TAB]
web01.localhost  -- web01 development server
web02.localhost  -- web02 development server
```

You can hide a host using `hidden` property. If you set it true, zsh completion doesn't show the host.

```lua
host "web01.localhost" {
    ForwardAgent = "yes",
    HostName = "192.168.0.11",
    Port = "22",
    User = "kohkimakimoto",
    description = "web01 development server",
    hidden = true,
}
```

You notice that the first characters of the `description` and `hidden` are lower case. Others are upper case. It is important point. Essh uses properties whose first character is upper case, as **SSH config** generated to the temporary file. And the properties whose first character is lower case are used for special purpose of Essh functions, not ssh config.

## Configuration

### Syntax

Essh configuration is written in [Lua](https://www.lua.org/). In the configuration files, you can also use DSL syntax that is more human-readable. Here is an example:

```lua
host "web01.localhost" {
    HostName = "192.168.0.11",
    Port = "22",
    User = "kohkimakimoto",
    description = "web01 development server",
    tags = {
        "web",
    },
}

host "web02.localhost" {
    HostName = "192.168.0.12",
    Port = "22",
    User = "kohkimakimoto",
    description = "web02 development server",
    tags = {
        "web",
    },
}

task "uptime" {
    on = "web",
    script = "uptime",
}
```

### Configuration files

Essh loads configuration files from three different places.

* At first, loads `/etc/essh/config.lua` that is the system-wide configuration.
* At second, loads `~/.essh/config.lua` that is the per-user configuration.
* At last, loads `essh.lua` in the current directory for the per-project configuration.

### Hosts

WIP...

#### Example

```lua
host "web01.localhost" {
    ForwardAgent = "yes",
    HostName = "192.168.0.11",
    Port = "22",
    User = "kohkimakimoto",
    description = "web01 development server",
    hidden = false,
    tags = {

    },
    hooks = {

    },
}
```

#### Special purpose properties

* `tags`: Tags classify hosts.

    ```lua
    tags = {
        "web",
        "production",
    }
    ```

* `description`: Description.
* `hidden`: If you set it true, zsh completion doesn't show the host.
* `hooks`: Hooks is a table that defines `before_connect`, `after_connect` and `after_disconnect`.

    ```lua
    hooks = {
        -- Runs the script on the local before connecting. This is an example to change screen color to red.
        before_connect = "osascript -e 'tell application \"Terminal\" to set current settings of first window to settings set \"Red Sands\"'",

        -- Runs the script on the remote after connecting.
        after_connect = [=[
        echo "Connected to $(hostname)"
        ]=],

        -- Runs the script on the local after disconnecting. This is an example to change screen color to black.
        after_disconnect = "osascript -e 'tell application \"Terminal\" to set current settings of first window to settings set \"Pro\"'",
    }
    ```

    `before_connect` and `after_disconnect` also can be written as Lua function instead of shell script.

### Tasks

WIP...

### Modules

WIP...

### Libraries

WIP...

### Drivers

WIP...

## Command line options

* `--version`: Print version.
* `--help`: Print help.
* `--print`: Print generated ssh config.

## Integrating other SSH related commands

Essh can be used with `scp`, `rsync` and `git`.

* `git`: To use Essh inside of the git command. Write the following line in your `~/.zshrc`.

    ```
    export GIT_SSH=essh
    ```

* `scp`: Essh support to use with scp.

  ```
  $ essh --scp <scp command args...>
  ```

  For more easy to use, you can run `eval "$(essh --aliases)"` in your `~/.zshrc`, the above code can be written as the following.

  ```
  $ escp <scp command args...>
  ```

* `rsync`: Essh support to use with rsync.

  ```
  $ essh --rsync <rsync command args...>
  ```

  For more easy to use, you can run `eval "$(essh --aliases)"` in your `~/.zshrc`, the above code can be written as the following.

  ```
  $ ersync <rsync command args...>
  ```

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
