package cobra

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
)

// GenFishCompletion generates fish completion and writes to the passed writer.
func (c *Command) GenFishCompletion(w io.Writer) error {
	buf := new(bytes.Buffer)

	writeFishPreamble(c, buf)
	writeFishCommandCompletion(c, c, buf)

	_, err := buf.WriteTo(w)
	return err
}

func writeFishPreamble(cmd *Command, buf *bytes.Buffer) {
	subCommandNames := []string{}
	rangeCommands(cmd, func(subCmd *Command) {
		subCommandNames = append(subCommandNames, subCmd.Name())
	})
	buf.WriteString(fmt.Sprintf(`
function __fish_%s_no_subcommand --description 'Test if oly has yet to be given the subcommand'
	for i in (commandline -opc)
		if contains -- $i %s
			return 1
		end
	end
	return 0
end
function __fish_%s_has_flag
  for i in (commandline -opc)
		if contains -- "--$1" $i
			return 0
		end
	end
	return 1
end
`, cmd.Name(), strings.Join(subCommandNames, " "), cmd.Name()))
}

func writeFishCommandCompletion(rootCmd, cmd *Command, buf *bytes.Buffer) {
	rangeCommands(cmd, func(subCmd *Command) {
		condition := commandCompletionCondition(rootCmd, cmd)
		buf.WriteString(fmt.Sprintf("complete -c %s -f %s -a %s -d '%s'\n", rootCmd.Name(), condition, subCmd.Name(), subCmd.Short))
	})
	writeCommandFlagsCompletion(rootCmd, cmd, buf)
	rangeCommands(cmd, func(subCmd *Command) {
		writeFishCommandCompletion(rootCmd, subCmd, buf)
	})
}

func writeCommandFlagsCompletion(rootCmd, cmd *Command, buf *bytes.Buffer) {
	cmd.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
		if nonCompletableFlag(flag) {
			return
		}
		writeCommandFlagCompletion(rootCmd, cmd, buf, flag)
	})
	cmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
		if nonCompletableFlag(flag) {
			return
		}
		writeCommandFlagCompletion(rootCmd, cmd, buf, flag)
	})
}

func writeCommandFlagCompletion(rootCmd, cmd *Command, buf *bytes.Buffer, flag *pflag.Flag) {
	shortHandPortion := ""
	if len(flag.Shorthand) > 0 {
		shortHandPortion = fmt.Sprintf("-s %s", flag.Shorthand)
	}
	condition := completionCondition(rootCmd, cmd)
	buf.WriteString(fmt.Sprintf("complete -c %s -f %s %s %s -l %s -d '%s'\n",
		rootCmd.Name(), condition, flagRequiresArgumentCompletion(flag), shortHandPortion, flag.Name, flag.Usage))
}

func flagRequiresArgumentCompletion(flag *pflag.Flag) string {
	if flag.Value.Type() != "bool" {
		return "-r"
	}
	return ""
}

func subCommandPath(rootCmd *Command, cmd *Command) string {
	path := []string{}
	currentCmd := cmd
	for {
		path = append([]string{currentCmd.Name()}, path...)
		if currentCmd.Parent() == rootCmd {
			return strings.Join(path, " ")
		}
		currentCmd = currentCmd.Parent()
	}
	return ""
}

func rangeCommands(cmd *Command, callback func(subCmd *Command)) {
	for _, subCmd := range cmd.Commands() {
		if !subCmd.IsAvailableCommand() || subCmd == cmd.helpCommand {
			continue
		}
		callback(subCmd)
	}
}

func commandCompletionCondition(rootCmd, cmd *Command) string {
	localNonPersistentFlags := cmd.LocalNonPersistentFlags()
	bareConditions := []string{}
	if rootCmd != cmd {
		bareConditions = append(bareConditions, fmt.Sprintf("__fish_seen_subcommand_from %s", subCommandPath(rootCmd, cmd)))
	} else {
		bareConditions = append(bareConditions, fmt.Sprintf("__fish_%s_no_subcommand", rootCmd.Name()))
	}
	localNonPersistentFlags.VisitAll(func(flag *pflag.Flag) {
		bareConditions = append(bareConditions, fmt.Sprintf("not __fish_%s_has_flag %s", rootCmd.Name(), flag.Name))
	})
	return fmt.Sprintf("-n '%s'", strings.Join(bareConditions, "; and "))
}

func completionCondition(rootCmd, cmd *Command) string {
	condition := fmt.Sprintf("-n '__fish_%s_no_subcommand'", rootCmd.Name())
	if rootCmd != cmd {
		condition = fmt.Sprintf("-n '__fish_seen_subcommand_from %s'", subCommandPath(rootCmd, cmd))
	}
	return condition
}
