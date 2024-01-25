package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Stack struct {
	Goroutine uint
	Reason    string
	Delay     string

	Calls   []*Call
	Creator *Creator

	Source []string
}

func (s *Stack) String() string {
	return strings.Join(s.Source, "\n")
}

type Call struct {
	Name string
	Args string
	*Location
}

type Creator struct {
	Name      string
	Goroutine uint
	*Location
}

type Location struct {
	Filename string
	Line     uint
}

func ParseStacks(dt string) ([]*Stack, error) {
	lines := strings.Split(dt, "\n")

	var stacks []*Stack
	var next []string
	for _, line := range lines {
		if len(line) != 0 {
			next = append(next, line)
			continue
		}
		if len(next) == 0 {
			continue
		}

		stack, err := parseStack(next)
		if err != nil {
			return nil, err
		}
		next = nil
		stacks = append(stacks, stack)
	}
	if len(next) > 0 {
		stack, err := parseStack(next)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, stack)
	}

	return stacks, nil
}

func parseStack(lines []string) (*Stack, error) {
	stack := Stack{
		Source: lines,
	}

	rest, ok := strings.CutPrefix(lines[0], "goroutine ")
	if !ok {
		return nil, fmt.Errorf("stack should begin with \"goroutine\":\n\t%s", lines[0])
	}
	ns, rest, _ := strings.Cut(rest, " ")

	n, err := strconv.ParseUint(ns, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse goroutine number: %w", err)
	}
	stack.Goroutine = uint(n)

	rest = strings.TrimSuffix(strings.TrimPrefix(rest, "["), "]:")
	if reason, delay, ok := strings.Cut(rest, ", "); ok {
		stack.Reason = reason
		stack.Delay = delay
	} else {
		stack.Reason = rest
	}

	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "traceback: unexpected SPWRITE function") {
			continue
		}

		if rest, ok := strings.CutPrefix(lines[i], "created by "); ok {
			creator := Creator{}

			name, goroutine, ok := strings.Cut(rest, " in goroutine ")
			if !ok {
				return nil, fmt.Errorf("creator line should a goroutine:\n\t%s", lines[i])
			}
			creator.Name = name

			goroutineNo, err := strconv.ParseUint(goroutine, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("could not parse goroutine number: %w", err)
			}
			creator.Goroutine = uint(goroutineNo)

			if i < len(lines)-1 {
				i++
				loc, err := parseLocation(lines[i])
				if err != nil {
					return nil, err
				}
				creator.Location = loc
			}

			stack.Creator = &creator
			continue
		}

		call := Call{}

		name, args, ok := strings.Cut(lines[i], "(")
		if !ok {
			return nil, fmt.Errorf("trace line should have a function:\n\t%s", lines[i])
		}
		args = strings.TrimSuffix(args, ")")

		call.Name = name
		call.Args = args

		if i < len(lines)-1 {
			i++
			loc, err := parseLocation(lines[i])
			if err != nil {
				return nil, err
			}
			call.Location = loc
		}

		stack.Calls = append(stack.Calls, &call)
	}

	return &stack, nil
}

func parseLocation(line string) (*Location, error) {
	location := Location{}

	line, ok := strings.CutPrefix(line, "\t")
	if !ok {
		return nil, fmt.Errorf("location line should be indented:\n\t%s", line)
	}
	// TODO: parse the other parts of this line
	parts := strings.Split(line, " ")

	if filename, line, ok := strings.Cut(parts[0], ":"); ok {
		lineNo, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse line number: %w", err)
		}

		location.Filename = filename
		location.Line = uint(lineNo)
	}

	return &location, nil
}
