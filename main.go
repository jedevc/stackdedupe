package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

var cmd = &cobra.Command{
	Args: cobra.ExactArgs(1),
	Use:  "stackdedupe <STACK DUMP>",
	RunE: func(cmd *cobra.Command, args []string) error {
		var stacks []*Stack
		for _, arg := range args {
			dt, err := os.ReadFile(arg)
			if err != nil {
				return err
			}

			nstacks, err := ParseStacks(string(dt))
			if err != nil {
				return err
			}
			stacks = append(stacks, nstacks...)

			fmt.Fprintf(os.Stderr, "[+] Imported %d stack traces from %s\n", len(stacks), arg)
		}

		stacksUniq := dedupeStacks(stacks)
		sort.Slice(stacksUniq, func(i, j int) bool {
			return stacksUniq[i].Goroutine < stacksUniq[j].Goroutine
		})
		fmt.Fprintf(os.Stderr, "[+] Found %d unique stack traces (removed %d)\n", len(stacksUniq), len(stacks)-len(stacksUniq))

		stacksFiltered := []*UniqStack{}
		for _, stack := range stacksUniq {
			if stack.Reason == "idle" || strings.Contains(stack.Reason, "(idle)") {
				continue
			}
			if strings.HasPrefix(stack.Reason, "GC ") {
				continue
			}
			if stack.Reason == "finalizer wait" {
				continue
			}
			stacksFiltered = append(stacksFiltered, stack)
		}
		fmt.Fprintf(os.Stderr, "[+] Found %d important stack traces (removed %d)\n", len(stacksFiltered), len(stacksUniq)-len(stacksFiltered))

		for _, stack := range stacksFiltered {
			fmt.Printf("(%d copies)\n", len(stack.Variants)+1)
			fmt.Println(stack.String())
			fmt.Println()
		}
		return nil
	},
}

type UniqStack struct {
	*Stack
	Variants []*Stack
}

func dedupeStacks(stacks []*Stack) []*UniqStack {
	m := make(map[string]*UniqStack)

	for _, stack := range stacks {
		var buff bytes.Buffer
		for _, call := range stack.Calls {
			fmt.Fprintf(&buff, "%s %s:%d\n", call.Name, call.Filename, call.Line)
		}
		k := buff.String()

		if _, ok := m[k]; !ok {
			m[k] = &UniqStack{Stack: stack}
		} else {
			m[k].Variants = append(m[k].Variants, stack)
		}
	}

	result := make([]*UniqStack, 0, len(m))
	for _, stack := range m {
		result = append(result, stack)
	}
	return result
}
