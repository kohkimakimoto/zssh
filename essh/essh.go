package essh

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/kohkimakimoto/essh/color"
	"github.com/kohkimakimoto/essh/helper"
	"github.com/yuin/gopher-lua"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/template"
)

// system configurations.
var (
	SystemWideConfigFile string
	PerUserConfigFile    string
	CurrentDirConfigFile string
)

// flags
var (
	versionFlag            bool
	helpFlag               bool
	printFlag              bool
	configFlag             bool
	systemConfigFlag       bool
	debugFlag              bool
	hostsFlag              bool
	quietFlag              bool
	tagsFlag               bool
	tasksFlag              bool
	genFlag                bool
	zshCompletionFlag      bool
	zshCompletionHostsFlag bool
	zshCompletionTagsFlag  bool
	zshCompletionTasksFlag bool
	bashCompletionFlag     bool
	aliasesFlag            bool
	fileFlag               bool
	execFlag               bool
	localExecFlag          bool
	noPrefixFlag           bool
	rsyncFlag              bool
	scpFlag                bool

	configFile string
	filters    []string = []string{}
	format     string
)

func Start() error {
	if len(os.Args) == 1 {
		printUsage()
		return nil
	}

	osArgs := os.Args[1:]
	args := []string{}

	for {
		if len(osArgs) == 0 {
			break
		}

		arg := osArgs[0]
		if arg == "--print" {
			printFlag = true
		} else if arg == "--version" {
			versionFlag = true
		} else if arg == "--help" {
			helpFlag = true
		} else if arg == "--config" {
			configFlag = true
		} else if arg == "--system-config" {
			systemConfigFlag = true
		} else if arg == "--debug" {
			debugFlag = true
		} else if arg == "--hosts" {
			hostsFlag = true
		} else if arg == "--quiet" {
			quietFlag = true
		} else if arg == "--tasks" {
			tasksFlag = true
		} else if arg == "--filter" {
			if len(osArgs) < 2 {
				return fmt.Errorf("--filter reguires an argument.")
			}
			filters = append(filters, osArgs[1])
			osArgs = osArgs[1:]
		} else if strings.HasPrefix(arg, "--filter=") {
			filters = append(filters, strings.Split(arg, "=")[1])
		} else if arg == "--format" {
			if len(osArgs) < 2 {
				return fmt.Errorf("--format reguires an argument.")
			}
			format = osArgs[1]
			osArgs = osArgs[1:]
		} else if strings.HasPrefix(arg, "--format=") {
			format = strings.Split(arg, "=")[1]
		} else if arg == "--tags" {
			tagsFlag = true
		} else if arg == "--gen" {
			genFlag = true
		} else if arg == "--zsh-completion" {
			zshCompletionFlag = true
		} else if arg == "--zsh-completion-hosts" {
			zshCompletionHostsFlag = true
		} else if arg == "--zsh-completion-tags" {
			zshCompletionTagsFlag = true
		} else if arg == "--zsh-completion-tasks" {
			zshCompletionTasksFlag = true
		} else if arg == "--bash-completion" {
			bashCompletionFlag = true
		} else if arg == "--aliases" {
			aliasesFlag = true
		} else if arg == "--config-file" {
			if len(osArgs) < 2 {
				return fmt.Errorf("--config-file reguires an argument.")
			}
			configFile = osArgs[1]
			osArgs = osArgs[1:]
		} else if strings.HasPrefix(arg, "--config-file=") {
			configFile = strings.Split(arg, "=")[1]
		} else if arg == "--exec" {
			execFlag = true
		} else if arg == "--local-exec" {
			localExecFlag = true
		} else if arg == "--no-prefix" {
			noPrefixFlag = true
		} else if arg == "--file" {
			fileFlag = true
		} else if arg == "--rsync" {
			rsyncFlag = true
		} else if arg == "--scp" {
			scpFlag = true
		} else if strings.HasPrefix(arg, "--") {
			return fmt.Errorf("invalid option '%s'.", arg)
		} else {
			// restructure args to remove essh options.
			args = append(args, arg)
		}

		osArgs = osArgs[1:]
	}

	if helpFlag {
		printHelp()
		return nil
	}

	if versionFlag {
		fmt.Printf("%s (%s)\n", Version, CommitHash)
		return nil
	}

	if zshCompletionFlag {
		fmt.Print(ZSH_COMPLETION)
		return nil
	}

	if aliasesFlag {
		fmt.Print(ALIASES_CODE)
		return nil
	}

	if bashCompletionFlag {
		fmt.Print(BASH_COMPLETION)
		return nil
	}

	if configFlag {
		runCommand("$EDITOR " + PerUserConfigFile)
		return nil
	}

	if systemConfigFlag {
		runCommand("$EDITOR " + SystemWideConfigFile)
		return nil
	}

	// set up the lua state.
	L := lua.NewState()
	defer L.Close()

	// init lua state
	InitLuaState(L)

	if debugFlag {
		fmt.Printf("[essh debug] init lua state\n")
	}

	// generate temporary ssh config file
	tmpFile, err := ioutil.TempFile("", "essh.ssh_config.")
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())

		if debugFlag {
			fmt.Printf("[essh debug] deleted config file: %s \n", tmpFile.Name())
		}

	}()
	temporarySSHConfigFile := tmpFile.Name()

	if debugFlag {
		fmt.Printf("[essh debug] generated config file: %s \n", temporarySSHConfigFile)
	}

	// set temporary ssh config file path
	lessh.RawSetString("ssh_config", lua.LString(temporarySSHConfigFile))

	// load specific config file
	if configFile != "" {
		_, err := os.Stat(configFile)
		if err != nil {
			return err
		}

		if err := L.DoFile(configFile); err != nil {
			return err
		}

		if debugFlag {
			fmt.Printf("[essh debug] loaded config file: %s \n", configFile)
		}

	} else {
		// load system wide config
		if _, err := os.Stat(SystemWideConfigFile); err == nil {
			if err := L.DoFile(SystemWideConfigFile); err != nil {
				return err
			}

			if debugFlag {
				fmt.Printf("[essh debug] loaded config file: %s \n", SystemWideConfigFile)
			}
		}

		// load per-user wide config
		if _, err := os.Stat(PerUserConfigFile); err == nil {
			if err := L.DoFile(PerUserConfigFile); err != nil {
				return err
			}

			if debugFlag {
				fmt.Printf("[essh debug] loaded config file: %s \n", PerUserConfigFile)
			}
		}

		// load current dir config
		if CurrentDirConfigFile != "" {
			if _, err := os.Stat(CurrentDirConfigFile); err == nil {
				if err := L.DoFile(CurrentDirConfigFile); err != nil {
					return err
				}

				if debugFlag {
					fmt.Printf("[essh debug] loaded config file: %s \n", CurrentDirConfigFile)
				}
			}
		}
	}

	if err := validateConfig(); err != nil {
		return err
	}

	// generate ssh hosts config
	content, err := GenHostsConfig()
	if err != nil {
		return err
	}

	// only print generated config
	if printFlag {
		fmt.Println(string(content))
		return nil
	}

	// show hosts for zsh completion
	if zshCompletionHostsFlag {
		for _, host := range Hosts {
			if !host.Hidden {
				fmt.Printf("%s\t%s\n", host.Name, host.Description)
			}
		}

		return nil
	}

	// show tasks for zsh completion
	if zshCompletionTasksFlag {
		for _, task := range Tasks {
			fmt.Printf("%s\t%s\n", task.Name, task.Description)
		}
		return nil
	}

	if zshCompletionTagsFlag {
		for _, tag := range Tags() {
			fmt.Printf("%s\n", tag)
		}
		return nil
	}

	// only print hosts list
	if hostsFlag {
		var hosts []*Host
		if len(filters) > 0 {
			hosts = HostsByNames(filters)
		} else {
			hosts = Hosts
		}

		if format == "json" {
			printJson(hosts, "")
		} else if format == "prettyjson" {
			printJson(hosts, "    ")
		} else {
			tb := helper.NewPlainTable(os.Stdout)
			if !quietFlag {
				tb.SetHeader([]string{"NAME", "DESCRIPTION", "TAGS"})
			}
			for _, host := range hosts {
				if !host.Hidden {
					if quietFlag {
						tb.Append([]string{host.Name})
					} else {
						tb.Append([]string{host.Name, host.Description, strings.Join(host.Tags, ",")})
					}
				}
			}
			tb.Render()
		}

		return nil
	}

	// only print tags list
	if tagsFlag {
		tb := helper.NewPlainTable(os.Stdout)
		if !quietFlag {
			tb.SetHeader([]string{"NAME"})
		}
		for _, tag := range Tags() {
			tb.Append([]string{tag})
		}
		tb.Render()

		return nil
	}

	if tasksFlag {
		tb := helper.NewPlainTable(os.Stdout)
		if !quietFlag {
			tb.SetHeader([]string{"NAME", "DESCRIPTION", "ON"})
		}
		for _, t := range Tasks {
			tb.Append([]string{t.Name, t.Description, strings.Join(t.On, ",")})
		}
		tb.Render()

		return nil
	}

	outputConfig, ok := toString(lessh.RawGetString("ssh_config"))
	if !ok {
		return fmt.Errorf("invalid value %v in the 'ssh_config'", lessh.RawGetString("ssh_config"))
	}

	if debugFlag {
		fmt.Printf("[essh debug] output ssh_config contents to the file: %s \n", outputConfig)
	}

	// update temporary ssh config file
	err = ioutil.WriteFile(outputConfig, content, 0644)
	if err != nil {
		return err
	}

	// only generating contents
	if genFlag {
		return nil
	}

	// select running mode and run it.
	if execFlag {
		if len(filters) == 0 {
			return fmt.Errorf("exec mode requires --filter option.")
		}

		hosts := HostsByNames(filters)

		if len(args) != 1 {
			return fmt.Errorf("exec mode requires 1 parameter that is the executed command string).")
		}

		err = runExec(outputConfig, args[0], hosts, noPrefixFlag, fileFlag)
	} else if localExecFlag {
		if len(filters) == 0 {
			return fmt.Errorf("local-exec mode requires --filter option.")
		}

		hosts := HostsByNames(filters)

		if len(args) != 1 {
			return fmt.Errorf("local-exec mode requires 1 parameter that is the executed command string).")
		}

		err = runLocalExec(outputConfig, args[0], hosts, noPrefixFlag, fileFlag)
	} else if rsyncFlag {
		err = runRsync(outputConfig, args)
	} else if scpFlag {
		err = runSCP(outputConfig, args)
	} else {
		// try to get a task.
		if len(args) > 0 {
			taskName := args[0]
			task := GetTask(taskName)
			if task != nil {
				if len(args) > 2 {
					return fmt.Errorf("too many arguments.")
				} else if len(args) == 2 {
					return runTask(outputConfig, task, args[1])
				} else {
					return runTask(outputConfig, task, "")
				}
			}
		}

		// run ssh command
		err = runSSH(outputConfig, args)
	}

	return err
}

func printJson(hosts []*Host, indent string) {
	convHosts := []map[string]map[string]interface{}{}

	for _, host := range hosts {
		h := map[string]map[string]interface{}{}

		hv := map[string]interface{}{}
		for _, pair := range host.Values() {
			for k, v := range pair {
				hv[k] = v
			}
		}
		h[host.Name] = hv

		hv["description"] = host.Description
		hv["Hidden"] = host.Hidden
		hv["tags"] = host.Tags

		convHosts = append(convHosts, h)
	}

	if indent == "" {
		b, err := json.Marshal(convHosts)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
	} else {
		b, err := json.MarshalIndent(convHosts, "", indent)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
	}
}

func runExec(config string, command string, hosts []*Host, noPrefixFlag bool, fileFlag bool) error {
	if debugFlag {
		hostNames := []string{}
		for _, h := range hosts {
			hostNames = append(hostNames, h.Name)
		}
		fmt.Printf("[essh debug] target hosts:%s\n", strings.Join(hostNames, ", "))
	}

	if fileFlag {
		// if the fileFlag is on. get a command from a file.
		b, err := getScriptContent(command)
		if err != nil {
			return err
		}
		command = string(b)
	}

	wg := &sync.WaitGroup{}
	m := new(sync.Mutex)
	for _, host := range hosts {
		wg.Add(1)
		go func(config string, command string, host *Host) {
			err := runRemoteScript(config, command, host, m, noPrefixFlag)
			if err != nil {
				fmt.Fprintf(color.StderrWriter, color.FgRB("[essh error] %v\n", err))
				panic(err)
			}

			wg.Done()
		}(config, command, host)
	}
	wg.Wait()

	return nil
}

func runLocalExec(config string, command string, hosts []*Host, noPrefixFlag bool, fileFlag bool) error {
	if debugFlag {
		hostNames := []string{}
		for _, h := range hosts {
			hostNames = append(hostNames, h.Name)
		}
		fmt.Printf("[essh debug] target hosts:%s\n", strings.Join(hostNames, ", "))
	}

	if fileFlag {
		// if the fileFlag is on. get a command from a file.
		b, err := getScriptContent(command)
		if err != nil {
			return err
		}
		command = string(b)
	}

	wg := &sync.WaitGroup{}
	m := new(sync.Mutex)
	for _, host := range hosts {
		wg.Add(1)
		go func(config string, command string, host *Host) {
			err := runLocalScript(config, command, host, m, noPrefixFlag)
			if err != nil {
				fmt.Fprintf(color.StderrWriter, color.FgRB("[essh error] %v\n", err))
				panic(err)
			}

			wg.Done()
		}(config, command, host)
	}
	wg.Wait()

	return nil
}

func runTask(config string, task *Task, payload string) error {
	if debugFlag {
		fmt.Printf("[essh debug] run task: %s\n", task.Name)
	}

	if task.Prepare != nil {
		if debugFlag {
			fmt.Printf("[essh debug] run prepare function.\n")
		}

		ctx := NewTaskContext(task, payload)
		err := task.Prepare(ctx)
		if err != nil {
			return err
		}

		payload = ctx.Payload
	}

	// get target hosts.
	on := task.On
	if len(on) > 0 {
		// run remotely.
		hosts := HostsByNames(on)
		wg := &sync.WaitGroup{}
		m := new(sync.Mutex)
		for _, host := range hosts {
			if task.Parallel {
				wg.Add(1)
				go func(config string, task *Task, payload string, host *Host) {
					err := runRemoteTaskScript(config, task, payload, host, m)
					if err != nil {
						fmt.Fprintf(color.StderrWriter, color.FgRB("[essh error] %v\n", err))
						panic(err)
					}

					wg.Done()
				}(config, task, payload, host)
			} else {
				err := runRemoteTaskScript(config, task, payload, host, m)
				if err != nil {
					return err
				}
			}
		}
		wg.Wait()
	} else {
		// run locally.
		err := runLocalTaskScript(task, payload)
		if err != nil {
			return err
		}
	}

	return nil
}

func runRemoteScript(config string, command string, host *Host, m *sync.Mutex, noPrefixFlag bool) error {
	// setup ssh command args
	var sshCommandArgs []string
	sshCommandArgs = []string{"-F", config, host.Name}

	var script string
	script = "export ESSH_HOSTNAME=" + host.Name + "\n" + command

	// inspired by https://github.com/laravel/envoy
	delimiter := "EOF-ESSH-SCRIPT"
	sshCommandArgs = append(sshCommandArgs, "bash", "-se", "<<\\"+delimiter+"\n"+script+"\n"+delimiter)

	cmd := exec.Command("ssh", sshCommandArgs[:]...)
	if debugFlag {
		fmt.Printf("[essh debug] real ssh command: %v \n", cmd.Args)
	}

	cmd.Stdin = os.Stdin

	prefix := ""
	if !noPrefixFlag {
		dict := map[string]interface{}{
			"Host": host,
		}
		tmpl, err := template.New("T").Parse("[{{.Host.Name}}] ")
		if err != nil {
			return err
		}
		var b bytes.Buffer
		err = tmpl.Execute(&b, dict)
		if err != nil {
			return err
		}

		prefix = b.String()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	// inspired by https://github.com/fujiwara/nssh/blob/master/nssh.go
	go scanLines(stdout, color.StdoutWriter, prefix, m)
	go scanLines(stderr, color.StderrWriter, prefix, m)

	return cmd.Wait()
}

func runLocalScript(config string, command string, host *Host, m *sync.Mutex, noPrefixFlag bool) error {
	var shell, flag string
	if runtime.GOOS == "windows" {
		shell = "cmd"
		flag = "/C"
	} else {
		shell = "/bin/sh"
		flag = "-c"
	}

	script := "export ESSH_HOSTNAME=" + host.Name + "\n" + command
	cmd := exec.Command(shell, flag, script)

	if debugFlag {
		fmt.Printf("[essh debug] real local command: %v \n", cmd.Args)
	}

	cmd.Stdin = os.Stdin

	prefix := ""
	if !noPrefixFlag {
		dict := map[string]interface{}{
			"Host": host,
		}
		tmpl, err := template.New("T").Parse("[{{.Host.Name}}] ")
		if err != nil {
			return err
		}
		var b bytes.Buffer
		err = tmpl.Execute(&b, dict)
		if err != nil {
			return err
		}

		prefix = b.String()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}
	// inspired by https://github.com/fujiwara/nssh/blob/master/nssh.go
	go scanLines(stdout, color.StdoutWriter, prefix, m)
	go scanLines(stderr, color.StderrWriter, prefix, m)
	return cmd.Wait()
}

func runRemoteTaskScript(config string, task *Task, payload string, host *Host, m *sync.Mutex) error {
	// setup ssh command args
	var sshCommandArgs []string
	if task.Tty {
		sshCommandArgs = []string{"-t", "-t", "-F", config, host.Name}
	} else {
		sshCommandArgs = []string{"-F", config, host.Name}
	}

	var script string
	if task.Privileged {
		script = "sudo sudo su - <<\\EOF-ESSH-PRIVILEGED\n export ESSH_PAYLOAD=" + ShellEscape(payload) + "\n" + task.Script + "\n" + "EOF-ESSH-PRIVILEGED"
	} else {
		script = "export ESSH_PAYLOAD=" + ShellEscape(payload) + "\n" + task.Script
	}

	// inspired by https://github.com/laravel/envoy
	delimiter := "EOF-ESSH-SCRIPT"
	sshCommandArgs = append(sshCommandArgs, "bash", "-se", "<<\\"+delimiter+"\n"+script+"\n"+delimiter)

	cmd := exec.Command("ssh", sshCommandArgs[:]...)
	if debugFlag {
		fmt.Printf("[essh debug] real ssh command: %v \n", cmd.Args)
	}

	cmd.Stdin = os.Stdin

	prefix := ""
	if !noPrefixFlag {
		dict := map[string]interface{}{
			"Host": host,
			"Task": task,
		}
		tmpl, err := template.New("T").Parse(task.Prefix)
		if err != nil {
			return err
		}
		var b bytes.Buffer
		err = tmpl.Execute(&b, dict)
		if err != nil {
			return err
		}

		prefix = b.String()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	// inspired by https://github.com/fujiwara/nssh/blob/master/nssh.go
	go scanLines(stdout, color.StdoutWriter, prefix, m)
	go scanLines(stderr, color.StderrWriter, prefix, m)

	return cmd.Wait()
}

func scanLines(src io.ReadCloser, dest io.Writer, prefix string, m *sync.Mutex) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		func(m *sync.Mutex) {
			m.Lock()
			defer m.Unlock()
			if prefix != "" {
				fmt.Fprintf(dest, "%s%s\n", color.FgCB(prefix), scanner.Text())
			} else {
				fmt.Fprintf(dest, "%s\n", scanner.Text())
			}
		}(m)
	}
}

func runLocalTaskScript(task *Task, payload string) error {
	var shell, flag string
	if runtime.GOOS == "windows" {
		shell = "cmd"
		flag = "/C"
	} else {
		shell = "/bin/sh"
		flag = "-c"
	}

	script := "export ESSH_PAYLOAD=" + payload + "\n" + task.Script
	cmd := exec.Command(shell, flag, script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if debugFlag {
		fmt.Printf("[essh debug] real local command: %v \n", cmd.Args)
	}

	return cmd.Run()
}

func runSSH(config string, args []string) error {
	// hooks
	var hooks map[string]interface{}

	// Limitation!
	// hooks fires only when the hostname is just specified.
	if len(args) == 1 {
		hostname := args[0]
		if host := GetHost(hostname); host != nil {
			hooks = host.Hooks
		}
	}

	// run before_connect hook
	if before := hooks["before_connect"]; before != nil {
		if debugFlag {
			fmt.Printf("[essh debug] run before_connect hook\n")
		}
		err := runHook(before)
		if err != nil {
			return err
		}
	} else if before := hooks["before"]; before != nil {
		// for backward compatibility
		if debugFlag {
			fmt.Printf("[essh debug] run before hook\n")
		}
		err := runHook(before)
		if err != nil {
			return err
		}
	}

	// register after_disconnect hook
	defer func() {
		// after hook
		if after := hooks["after_disconnect"]; after != nil {
			if debugFlag {
				fmt.Printf("[essh debug] run after_disconnect hook\n")
			}
			err := runHook(after)
			if err != nil {
				panic(err)
			}
		} else if after := hooks["after"]; after != nil {
			// for backward compatibility
			if debugFlag {
				fmt.Printf("[essh debug] run after hook\n")
			}
			err := runHook(after)
			if err != nil {
				panic(err)
			}
		}
	}()

	// setup ssh command args
	var sshCommandArgs []string

	// run after_connect hook
	if afterConnect := hooks["after_connect"]; afterConnect != nil {
		sshCommandArgs = []string{"-t", "-F", config}
		sshCommandArgs = append(sshCommandArgs, args[:]...)

		script := afterConnect.(string)
		script += "\nexec $SHELL\n"

		sshCommandArgs = append(sshCommandArgs, script)

	} else {
		sshCommandArgs = []string{"-F", config}
		sshCommandArgs = append(sshCommandArgs, args[:]...)
	}

	// execute ssh commmand
	cmd := exec.Command("ssh", sshCommandArgs[:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if debugFlag {
		fmt.Printf("[essh debug] real ssh command: %v \n", cmd.Args)
	}

	return cmd.Run()
}

func runHook(hook interface{}) error {
	if hookFunc, ok := hook.(func() error); ok {
		err := hookFunc()
		if err != nil {
			return err
		}
	} else if hookString, ok := hook.(string); ok {
		err := runCommand(hookString)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid type hook: %v", hook)
	}
	return nil
}

func getScriptContent(shellPath string) ([]byte, error) {
	var scriptContent []byte
	if strings.HasPrefix(shellPath, "http://") || strings.HasPrefix(shellPath, "https://") {
		// get script from remote using http.
		if debugFlag {
			fmt.Printf("[essh debug] get script using http from '%s'\n", shellPath)
		}

		var httpClient *http.Client
		if strings.HasPrefix(shellPath, "https://") {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			httpClient = &http.Client{Transport: tr}
		} else {
			httpClient = &http.Client{}
		}

		resp, err := httpClient.Get(shellPath)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		scriptContent = b
	} else {
		// get script from the file system.
		b, err := ioutil.ReadFile(shellPath)
		if err != nil {
			return nil, err
		}
		scriptContent = b
	}

	return scriptContent, nil
}

func runSCP(config string, args []string) error {
	if debugFlag {
		fmt.Printf("[essh debug] use scp mode.\n")
	}

	if len(args) < 2 {
		return fmt.Errorf("scp mode requires 2 parameters at least.")
	}

	// In the scp mode.
	// the arguments must be scp command options and args.
	sshCommandArgs := []string{"-F", config}
	sshCommandArgs = append(sshCommandArgs, args[:]...)

	// execute ssh commmand
	cmd := exec.Command("scp", sshCommandArgs[:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if debugFlag {
		fmt.Printf("[essh debug] real ssh command: %v \n", cmd.Args)
	}

	return cmd.Run()
}

func runRsync(config string, args []string) error {
	if debugFlag {
		fmt.Printf("[essh debug] use rsync mode.\n")
	}

	if len(args) < 1 {
		return fmt.Errorf("rsync mode requires 1 parameters at least.")
	}

	// In the rsync mode.
	// the arguments must be rsync command options and args.
	sshCommandArgs := []string{"-F", config}
	rsyncSSHOption := `-e "ssh ` + strings.Join(sshCommandArgs, " ") + `"`

	rsyncCommand := "rsync " + rsyncSSHOption + " " + strings.Join(args, " ")

	if debugFlag {
		fmt.Printf("[essh debug] real rsync command: %v\n", rsyncCommand)
	}

	return runCommand(rsyncCommand)
}

func runCommand(command string) error {
	var shell, flag string
	if runtime.GOOS == "windows" {
		shell = "cmd"
		flag = "/C"
	} else {
		shell = "/bin/sh"
		flag = "-c"
	}
	cmd := exec.Command(shell, flag, command)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func validateConfig() error {
	// check duplication of the host, task and tag names
	names := map[string]bool{}
	for _, host := range Hosts {
		if _, ok := names[host.Name]; ok {
			return fmt.Errorf("'%s' is duplicated", host.Name)
		}
		names[host.Name] = true
	}

	for _, task := range Tasks {
		if _, ok := names[task.Name]; ok {
			return fmt.Errorf("'%s' is duplicated", task.Name)
		}
		names[task.Name] = true
	}

	for _, tag := range Tags() {
		if _, ok := names[tag]; ok {
			return fmt.Errorf("'%s' is duplicated", tag)
		}
		names[tag] = true
	}

	return nil
}

type CallbackWriter struct {
	Func func(data []byte)
}

func (w *CallbackWriter) Write(data []byte) (int, error) {
	if w.Func != nil {
		w.Func(data)
	}
	return len(data), nil
}

func printHelp() {
	printUsage()
	fmt.Print(`Running rsyc:
  You can use essh config for rsync using --rsync option.
  Syntax:

    essh --rsync <rsync options and args...>

  Examples:

    essh --rsync -avz /local/dir/ web01.localhost:/path/to/remote/dir

Running scp:
  You can use essh config for scp using --scp option.
  Syntax:

    essh --scp <scp options and args...>

  Examples:

    essh --scp web01.localhost:/path/to/file ./local/file

See also:
  ssh, rsync, scp

`)
}

func printUsage() {
	fmt.Print(`Usage: essh [<options>] [<ssh options and args...>]

  essh is an extended ssh command.
  version ` + Version + ` (` + CommitHash + `)

  Copyright (c) Kohki Makimoto <kohki.makimoto@gmail.com>
  The MIT License (MIT)

Options:
  --version               Print version.
  --help                  Print help.
  --print                 Print generated ssh config.
  --gen                   Only generating ssh config.
  --config                Edit per-user config file.
  --system-config         Edit system wide config file.
  --config-file <file>    Load configuration from the specific file.
                          If you use this option, it does not use other default config files like a "/etc/essh/config.lua".

  --hosts                 List hosts.
  --tags                  List tags.
  --quiet                 (Using with --hosts or --tags option) Show only names.
  --format <format>       (Using with --hosts or --tags option) Output specified format (json|prettyjson)
  --filter <tag|host>     (Using with --hosts option) Use only the hosts filtered with a tag or a host.

  --tasks                 List tasks.

  --zsh-completion        Output zsh completion code.
  --aliases               Output aliases code.

  --debug                 Output debug log.

  --exec                  Execute a command on the remote hosts in parallel.
  --local-exec            Execute a command on the local host in parallel.
  --filter <tag|host>     (Using with --exec or --local-exec option) Use only the hosts filtered with a tag or a host.
  --no-prefix             (Using with --exec or --local-exec option) Disable outputing hostname prefix.
  --file                  (Using with --exec or --local-exec option) Load commands from a file.

  --rsync                 Run rsync with essh configuration.
  --scp                   Run scp with essh configuration.

`)
}

func init() {
	if SystemWideConfigFile == "" {
		SystemWideConfigFile = "/etc/essh/config.lua"
	}
	if PerUserConfigFile == "" {
		home := userHomeDir()
		PerUserConfigFile = filepath.Join(home, ".essh/config.lua")
	}

	if CurrentDirConfigFile == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Printf("couldn't get working dir %v\n", err)
		} else {
			CurrentDirConfigFile = filepath.Join(wd, ".essh.lua")
		}
	}
}

var ZSH_COMPLETION = `# This is zsh completion code.
# If you want to use it. write the following code in your '.zshrc'
#   eval "$(essh --zsh-completion)"
_essh_hosts() {
    local -a __essh_hosts
    PRE_IFS=$IFS
    IFS=$'\n'
    __essh_hosts=($(essh --zsh-completion-hosts | awk -F'\t' '{print $1":"$2}'))
    IFS=$PRE_IFS
    _describe -t host "host" __essh_hosts
}

_essh_tasks() {
    local -a __essh_tasks
    PRE_IFS=$IFS
    IFS=$'\n'
    __essh_tasks=($(essh --zsh-completion-tasks | awk -F'\t' '{print $1":"$2}'))
    IFS=$PRE_IFS
    _describe -t task "task" __essh_tasks
}

_essh_tags() {
    local -a __essh_tags
    PRE_IFS=$IFS
    IFS=$'\n'
    __essh_tags=($(essh --zsh-completion-tags))
    IFS=$PRE_IFS
    _describe -t tag "tag" __essh_tags
}


_essh_options() {
    local -a __options
    __essh_options=(
        '--version:Print version.'
        '--help:Print help.'
        '--print:Print generated ssh config.'
        '--gen:Only generating ssh config.'
        '--config:Edit per-user config file.'
        '--system-config:Edit system wide config file.'
        '--config-file:Load configuration from the specific file.'
        '--hosts:List hosts.'
        '--tags:List tags.'
        '--quiet:Show only names.'
        '--format:Output specified format (json|prettyjson)'
        '--filter:Use only the hosts filtered with a tag or a host'
        '--tasks:List tasks.'
        '--debug:Output debug log.'
        '--exec:Execute a command on the remote hosts.'
        '--local-exec:Execute a command on the local host.'
        '--no-prefix:Disable outputing hostname prefix.'
        '--file:Load commands from a file.'
        '--rsync:Run rsync with essh configuration.'
        '--scp:Run scp with essh configuration.'
        '--zsh-completion:Output zsh completion code.'
        '--aliases:Output aliases code.'
     )
    _describe -t option "option" __essh_options
}

_essh () {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments \
        '1: :->all' \
        '*: :->args' \
		&& ret=0

    case $state in
        all)
            _essh_tasks
            _essh_hosts
            _essh_options
            ;;
        args)
            last_arg="${words[${#words[@]}-1]}"
            case $last_arg in
                --tags|--tasks|--print|--help|--version|--gen|--config|--system-config)
                    ;;
                --file|--config-file)
                    _files
                    ;;
                --exec|--local-exec)
                    _essh_options
                    ;;
                --hosts)
                    _essh_options
                    ;;
                --filter)
                    _essh_hosts
                    _essh_tags
                    ;;
                *)
                    _essh_tasks
                    _essh_hosts
                    _essh_options
                    _files
                    ;;
            esac
            ;;
        *)
            _essh_tasks
            _essh_hosts
            _essh_options
            _files
            ;;
    esac
}

compdef _essh essh
`

var ALIASES_CODE = `# This is aliaes code.
# If you want to use it. write the following code in your '.zshrc'
#   eval "$(essh --aliases)"
alias escp='essh --scp'
alias ersync='essh --rsync'
`

var BASH_COMPLETION = `
_essh_targets() {

}

_essh () {

}

complete -F _essh essh

`

//
//func runShellScript(config string, args []string) error {
//	if len(args) < 2 {
//		return fmt.Errorf("shell script mode requires 2 parameters at least.")
//	}
//
//	// In the shell script mode.
//	// the last argument must be a script file path.
//	shellPath := args[len(args)-1]
//	// remove it
//	args = args[:len(args)-1]
//
//	var scriptContent []byte
//	if strings.HasPrefix(shellPath, "http://") || strings.HasPrefix(shellPath, "https://") {
//		// get script from remote using http.
//		if debugFlag {
//			fmt.Printf("[essh debug] get script using http from '%s'\n", shellPath)
//		}
//
//		var httpClient *http.Client
//		if strings.HasPrefix(shellPath, "https://") {
//			tr := &http.Transport{
//				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
//			}
//			httpClient = &http.Client{Transport: tr}
//		} else {
//			httpClient = &http.Client{}
//		}
//
//		resp, err := httpClient.Get(shellPath)
//		if err != nil {
//			return err
//		}
//		defer resp.Body.Close()
//		b, err := ioutil.ReadAll(resp.Body)
//		if err != nil {
//			return err
//		}
//
//		scriptContent = b
//	} else {
//		// get script from the file system.
//		b, err := ioutil.ReadFile(shellPath)
//		if err != nil {
//			return err
//		}
//		scriptContent = b
//	}
//
//	if debugFlag {
//		fmt.Printf("[essh debug] script:\n%s\n", string(scriptContent))
//	}
//
//	// setup ssh command args
//	sshCommandArgs := []string{"-F", config}
//	sshCommandArgs = append(sshCommandArgs, args[:]...)
//
//	delimiter := "EOF-ESSH-SCRIPT"
//	sshCommandArgs = append(sshCommandArgs, "bash", "-se", "<<",
//		`\`+delimiter+"\n"+string(scriptContent)+"\n"+delimiter)
//
//	cmd := exec.Command("ssh", sshCommandArgs[:]...)
//	cmd.Stdin = os.Stdin
//	cmd.Stdout = os.Stdout
//	cmd.Stderr = os.Stderr
//
//	if debugFlag {
//		fmt.Printf("[essh debug] real ssh command: %v \n", cmd.Args)
//	}
//
//	return cmd.Run()
//}