package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/donnie-ellis/aop/pkg/types"
)

// WriteInventory serialises a slice of InventoryHost into an INI-format
// Ansible inventory file at path.
//
// Format produced:
//
//	[all]
//	hostname ansible_host=hostname var1=val1 ...
//
//	[group_name]
//	hostname
//	...
func WriteInventory(path string, hosts []types.InventoryHost) error {
	var sb strings.Builder

	sb.WriteString("[all]\n")
	for _, h := range hosts {
		sb.WriteString(h.Hostname)
		sb.WriteString(" ansible_host=")
		sb.WriteString(h.Hostname)
		for k, v := range h.Vars {
			fmt.Fprintf(&sb, " %s=%s", k, v)
		}
		sb.WriteByte('\n')
	}
	sb.WriteByte('\n')

	// Collect group membership.
	groups := map[string][]string{}
	for _, h := range hosts {
		for _, g := range h.Groups {
			groups[g] = append(groups[g], h.Hostname)
		}
	}
	for group, members := range groups {
		fmt.Fprintf(&sb, "[%s]\n", group)
		for _, m := range members {
			sb.WriteString(m)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
