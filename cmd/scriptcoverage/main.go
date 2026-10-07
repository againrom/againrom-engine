// Command scriptcoverage proves how the shipped campaign's authored script
// nodes cross the production builder, script pass, dispatch and claim-backed
// effect boundaries. It reads a lawful install and writes no game data.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

func main() {
	if err := runMain(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "scriptcoverage:", err)
		os.Exit(1)
	}
}

func runMain(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("scriptcoverage", flag.ContinueOnError)
	assets := fs.String("assets", "", "lawful game install root (or AGAINROM_ASSETS)")
	summary := fs.Bool("summary", false, "print only the denominator summary")
	ticks := fs.Int("ticks", 4000, "ordinary headless ticks to drive on every campaign map")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := *assets
	if root == "" {
		root = os.Getenv("AGAINROM_ASSETS")
	}
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	result, maps, err := loadCoverage(root)
	if err != nil {
		return err
	}
	if err := validateDenominator(result); err != nil {
		return err
	}
	if err := runControlled(result, maps); err != nil {
		return err
	}
	if err := runSynthetic(result, maps); err != nil {
		return err
	}
	if err := runNatural(result, maps, *ticks); err != nil {
		return err
	}
	if *summary {
		_, err := fmt.Fprintf(out, "root=%s maps=%d checks=%d instants=%d triggers=%d operation_rows=%d synthetic_rows=%d natural_exact_instants=%d\n",
			result.Root, result.Maps, result.Checks, result.Instants, result.Triggers, result.Reachable, len(result.Synthetic), result.Natural)
		return err
	}
	return writeCSV(out, result)
}

func writeCSV(out io.Writer, result *coverageResult) error {
	w := csv.NewWriter(out)
	if err := w.Write([]string{"row", "root", "map", "mission", "family", "node", "opcode", "subcommand", "node_count", "natural_count", "natural_changed", "natural_no_change",
		"present", "accepted", "reachable", "natural", "controlled", "dispatched", "effect",
		"disposition", "divergence", "source", "seed", "oracle", "claims"}); err != nil {
		return err
	}
	for _, r := range result.Nodes {
		sub := ""
		if r.HasSubCommand {
			sub = strconv.FormatInt(int64(r.SubCommand), 10)
		}
		fields := []string{"node", r.Root, r.Map, strconv.Itoa(r.Mission), r.Family, strconv.Itoa(r.Node),
			strconv.FormatInt(int64(r.Opcode), 10), sub, "1", boolCount(r.Natural), boolCount(r.NaturalChanged), boolCount(r.NaturalNoChange),
			strconv.FormatBool(r.Present), strconv.FormatBool(r.Accepted), strconv.FormatBool(r.Reachable),
			strconv.FormatBool(r.Natural), strconv.FormatBool(r.Controlled), strconv.FormatBool(r.Dispatched),
			strconv.FormatBool(r.Effect), r.Disposition, r.Divergence, r.Source, r.Seed, r.Oracle, r.Claims}
		if err := w.Write(fields); err != nil {
			return err
		}
	}
	for _, r := range result.Operations {
		fields := []string{"operation", r.Root, "", "", r.Family, "", strconv.FormatInt(int64(r.Opcode), 10), "",
			strconv.Itoa(r.Nodes), strconv.Itoa(r.Natural), strconv.Itoa(r.NaturalChanged), strconv.Itoa(r.NaturalNoChange),
			"true", "true", "true", strconv.FormatBool(r.Natural > 0),
			strconv.FormatBool(r.Controlled), strconv.FormatBool(r.Dispatched), strconv.FormatBool(r.Effect), r.Disposition, r.Divergence, r.Witness,
			r.Seed, r.Oracle, r.Claims}
		if err := w.Write(fields); err != nil {
			return err
		}
	}
	for _, r := range result.Synthetic {
		fields := []string{"synthetic", r.Root, r.Map, strconv.Itoa(r.Mission), r.Family, strconv.Itoa(r.Node),
			strconv.FormatInt(int64(r.Opcode), 10), strconv.FormatInt(int64(r.SubCommand), 10),
			"1", "0", "0", "0", strconv.FormatBool(r.Present), strconv.FormatBool(r.Accepted), "false", "false",
			strconv.FormatBool(r.Controlled), strconv.FormatBool(r.Dispatched), strconv.FormatBool(r.Effect), r.Disposition, r.Divergence,
			r.Case, r.Seed, r.Oracle, r.Claims}
		if err := w.Write(fields); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func boolCount(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
