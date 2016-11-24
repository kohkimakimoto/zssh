package essh

import (
	"bytes"
	"github.com/yuin/gopher-lua"
	"sort"
	"strings"
	"text/template"
)

type Host struct {
	Name                 string
	Description          string
	Props                map[string]string
	HooksBeforeConnect   []interface{}
	HooksAfterConnect    []interface{}
	HooksAfterDisconnect []interface{}
	Hidden               bool
	Tags                 []string
	Registry             *Registry
	Private              bool
	SSHConfig            map[string]string
	LValues              map[string]lua.LValue
}

func NewHost() *Host {
	return &Host{
		Props:                map[string]string{},
		HooksBeforeConnect:   []interface{}{},
		HooksAfterConnect:    []interface{}{},
		HooksAfterDisconnect: []interface{}{},
		Tags:                 []string{},
		SSHConfig:            map[string]string{},
		LValues:              map[string]lua.LValue{},
	}
}

//
// Spec note about Hosts (it is a little complicated!):
//   Hosts are stored into a space: "global" or "local" which are called as 'registry' (or 'context').
//   The registry is determined by a place of the configuration file that hosts are defined in.
//
//   Example:
//     /etc/essh/config.lua              -> "global"
//     ~/.essh/config.lua                -> "global"
//     /path/to/project/esshconfig.lua   -> "local"
//
//   Hosts also have configuration "scope". There are two types of scope: "public" and "private".
//
//   There are some rules about operating hosts.
//     * Each public hosts must be unique. (You can NOT define public hosts by the same name in the local and global registry.)
//     * Any hosts must be unique in a same registry. (You can NOT define hosts by the same name in the same registry.)
//     * Hosts used by task must be defined in a same registry. (Tasks can refer to only hosts defined in the same registry.)
//     * Private hosts is only used by tasks.
//     * There can be duplicated hosts in the entire registries. (You can define private hosts even if you define same name public hosts.)
//

func (h *Host) SortedSSHConfig() []map[string]string {
	values := []map[string]string{}

	var names []string

	for name, _ := range h.SSHConfig {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		v := h.SSHConfig[name]
		value := map[string]string{name: v}
		values = append(values, value)
	}

	return values
}

func (h *Host) DescriptionOrDefault() string {
	if h.Description == "" {
		return h.Name + " host"
	}

	return h.Description
}

func (h *Host) Scope() string {
	if h.Private {
		return "private"
	} else {
		return "public"
	}
}

func GetPublicHost(hostname string) *Host {
	for _, h := range NewHostQuery().GetPublicHostsOrderByName() {
		if h.Name == hostname {
			return h
		}
	}

	return nil
}

var hostsTemplate = `{{range $i, $host := .Hosts -}}
Host {{$host.Name}}{{range $ii, $param := $host.SortedSSHConfig}}{{range $k, $v := $param}}
    {{$k}} {{$v}}{{end}}{{end}}

{{end -}}`

func GenHostsConfig(enabledHosts []*Host) ([]byte, error) {
	tmpl, err := template.New("T").Parse(hostsTemplate)
	if err != nil {
		return nil, err
	}

	input := map[string]interface{}{"Hosts": enabledHosts}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, input); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func Tags() []string {
	tagsMap := map[string]string{}
	tags := []string{}

	for _, host := range NewHostQuery().GetHostsOrderByName() {
		for _, t := range host.Tags {
			if _, exists := tagsMap[t]; !exists {
				tagsMap[t] = t
				tags = append(tags, t)
			}
		}
	}

	sort.Strings(tags)

	return tags
}

func HostnameAlignString(host *Host, hosts []*Host) func(string) string {
	var maxlen int
	for _, h := range hosts {
		size := len(h.Name)
		if maxlen < size {
			maxlen = size
		}
	}

	var namelen = len(host.Name)
	return func(s string) string {
		diff := maxlen - namelen
		return strings.Repeat(s, 1+diff)
	}
}
