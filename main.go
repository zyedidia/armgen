package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	InstrGeneral   = "general"
	InstrSystem    = "system"
	InstrFloat     = "float"
	InstrFpSimd    = "fpsimd"
	InstrAdvSimd   = "advsimd"
	InstrSve       = "sve"
	InstrSve2      = "sve2"
	InstrMortlach  = "mortlach"
	InstrMortlach2 = "mortlach2"
)

var InstrBase = strings.Join([]string{
	InstrGeneral,
	InstrFloat,
	InstrFpSimd,
	InstrAdvSimd,
}, ",")

type DocVar struct {
	XMLName xml.Name `xml:"docvar"`
	Key     string   `xml:"key,attr"`
	Val     string   `xml:"value,attr"`
}

type DocVars struct {
	XMLName xml.Name `xml:"docvars"`
	Vars    []DocVar `xml:"docvar"`
}

func (d DocVars) AliasMnemonic() string {
	for _, v := range d.Vars {
		if v.Key == "alias_mnemonic" {
			return v.Val
		}
	}
	return ""
}

func (d DocVars) Mnemonic() string {
	for _, v := range d.Vars {
		if v.Key == "mnemonic" {
			return v.Val
		}
	}
	return ""
}

func (d DocVars) InstrClass() string {
	for _, v := range d.Vars {
		if v.Key == "instr-class" {
			return v.Val
		}
	}
	return ""
}

type PsText struct {
	XMLName xml.Name `xml:"pstext"`
	Content string   `xml:",innerxml"`
}

type Ps struct {
	XMLName xml.Name `xml:"ps"`
	Name    string   `xml:"name,attr"`
	PsText  PsText   `xml:"pstext"`
}

type PsSection struct {
	XMLName xml.Name `xml:"ps_section"`
	Ps      Ps
}

type ArchVariant struct {
	XMLName xml.Name `xml:"arch_variant"`
	Name    string   `xml:"name,attr"`
	Feature string   `xml:"feature,attr"`
}

type ArchVariants struct {
	XMLName  xml.Name      `xml:"arch_variants"`
	Variants []ArchVariant `xml:"arch_variant"`
}

type BitC struct {
	XMLName xml.Name `xml:"c"`
	Cols    int      `xml:"colspan,attr"`
	Value   string   `xml:",chardata"`
}

type Box struct {
	XMLName xml.Name `xml:"box"`
	Bits    []BitC   `xml:"c"`
	Name    string   `xml:"name,attr"`
}

type RegDiagram struct {
	XMLName xml.Name `xml:"regdiagram"`
	Name    string   `xml:"psname,attr"`
	Boxes   []Box    `xml:"box"`
}

func (r RegDiagram) String() string {
	b := &bytes.Buffer{}
	for i, box := range r.Boxes {
		if box.Name != "" {
			fmt.Fprintf(b, "%s=", box.Name)
		}
		for _, bit := range box.Bits {
			if bit.Value == "" {
				bit.Value = "x"
			}
			if bit.Cols == 0 {
				bit.Cols = 1
			}
			fmt.Fprint(b, strings.Repeat(bit.Value, bit.Cols))
		}
		if i != len(r.Boxes)-1 {
			fmt.Fprint(b, "|")
		}
	}
	return b.String()
}

type Encoding struct {
	XMLName xml.Name `xml:"encoding"`
	Docs    DocVars  `xml:"docvars"`
}

type IClass struct {
	XMLName      xml.Name     `xml:"iclass"`
	Name         string       `xml:"name,attr"`
	Id           string       `xml:"id,attr"`
	RegDiagram   RegDiagram   `xml:"regdiagram"`
	ArchVariants ArchVariants `xml:"arch_variants"`
	Code         PsSection    `xml:"ps_section"`
	Encodings    []Encoding   `xml:"encoding"`
}

type Classes struct {
	XMLName xml.Name `xml:"classes"`
	IClass  []IClass `xml:"iclass"`
}

type InsnSection struct {
	XMLName xml.Name  `xml:"instructionsection"`
	Docs    DocVars   `xml:"docvars"`
	Type    string    `xml:"type,attr"`
	Id      string    `xml:"id,attr"`
	Classes Classes   `xml:"classes"`
	Code    PsSection `xml:"ps_section"`
}

type Assign struct {
	Dest string
	Src  string
}

func getAssigns(code string) []Assign {
	lines := strings.Split(code, "\n")
	var assigns []Assign
	for _, l := range lines {
		before, after, found := strings.Cut(l, " = ")
		if !found {
			continue
		}
		assigns = append(assigns, Assign{
			Dest: before,
			Src:  after,
		})
	}
	return assigns
}

var linkrx = regexp.MustCompile(`<a.*?>`)

func (is InsnSection) ReadSet() []string {
	lines := strings.Split(is.Code.Ps.PsText.Content, "\n")
	var set []string
	for _, l := range lines {
		if strings.Contains(l, "impl-aarch64.X.read.2") {
			l = strings.ReplaceAll(l, "</a>", "")
			l = linkrx.ReplaceAllLiteralString(l, "")
			_, after, found := strings.Cut(l, "=")
			if !found {
				log.Fatal("no = in ", l)
			}
			set = append(set, strings.TrimSpace(after))
		}
	}
	return set
}

func (is InsnSection) Uses(op string) bool {
	lines := strings.Split(is.Code.Ps.PsText.Content, "\n")
	for _, l := range lines {
		if strings.Contains(l, op) {
			return true
		}
	}
	return false
}

func (is InsnSection) UsesPc() bool {
	return is.Uses("impl-aarch64.PC.read.0")
}

func (is InsnSection) ReadsMem() bool {
	return is.Uses("impl-aarch64.Mem.read.3")
}

func (is InsnSection) WritesMem() bool {
	return is.Uses("impl-aarch64.Mem.write.3")
}

func (is InsnSection) MemAtomic() bool {
	return is.Uses("impl-aarch64.MemAtomic.4")
}

func (is InsnSection) IsBranch() bool {
	return is.Uses("impl-shared.BranchTo.3")
}

func (is InsnSection) BaseVariant() bool {
	for _, c := range is.Classes.IClass {
		if len(c.ArchVariants.Variants) != 0 {
			return false
		}
	}
	return true
}

func (is InsnSection) Variant(version string) bool {
	for _, c := range is.Classes.IClass {
		for _, v := range c.ArchVariants.Variants {
			if v.Name == version {
				return true
			}
		}
	}
	return false
}

func (is InsnSection) VariantLE(version string) bool {
	for _, c := range is.Classes.IClass {
		if len(c.ArchVariants.Variants) == 0 {
			return true
		}
		for _, v := range c.ArchVariants.Variants {
			if strings.Compare(v.Name, version) <= 0 {
				return true
			}
		}
	}
	return false
}

func (is InsnSection) HasClass(class string) bool {
	iclass := "iclass_" + class
	for _, c := range is.Classes.IClass {
		if iclass == c.Id {
			return true
		}
	}
	if is.Docs.InstrClass() != "" && is.Docs.InstrClass() == class {
		return true
	}
	return false
}

func (is InsnSection) BaseArch() bool {
	if !strings.Contains(InstrBase, is.Docs.InstrClass()) {
		return false
	}
	return is.BaseVariant()
}

func (is InsnSection) MatchesArch(name string, feature string) bool {
	for _, c := range is.Classes.IClass {
		for _, v := range c.ArchVariants.Variants {
			if strings.Contains(v.Name, name) && strings.Contains(v.Feature, feature) {
				return true
			}
		}
	}
	return false
}

func (is InsnSection) WriteSet() []string {
	lines := strings.Split(is.Code.Ps.PsText.Content, "\n")
	set := make(map[string]bool)
	for _, l := range lines {
		if strings.Contains(l, "impl-aarch64.X.write.2") {
			l = strings.ReplaceAll(l, "</a>", "")
			l = linkrx.ReplaceAllLiteralString(l, "")
			before, _, found := strings.Cut(l, "=")
			if !found {
				log.Fatal("no = in ", l)
			}
			set[strings.TrimSpace(before)] = true
		}
	}
	var ret []string
	for k := range set {
		ret = append(ret, k)
	}
	return ret
}

func (is InsnSection) Names() []string {
	set := make(map[string]bool)
	for _, c := range is.Classes.IClass {
		for _, e := range c.Encodings {
			set[e.Docs.Mnemonic()] = true
			alias := e.Docs.AliasMnemonic()
			if alias != "" {
				set[e.Docs.AliasMnemonic()] = true
			}
		}
	}
	var names []string
	for k := range set {
		names = append(names, k)
	}
	return names
}

func main() {
	base := flag.Bool("base", true, "only consider instructions from the ARMv8.0 instruction set")
	classes := flag.String("classes", "general,float,fpsimd,advsimd", "comma-separated list of instruction classes")
	branch := flag.Bool("branch", false, "only show branch instructions")
	rdmem := flag.Bool("rdmem", false, "only show instructions that read from memory")
	wrmem := flag.Bool("wrmem", false, "only show instructions that write to memory")
	atomic := flag.Bool("atomic", false, "only show atomic instructions")
	nomodify := flag.Bool("nomodify", false, "only show instructions that do not modify any general-purpose registers")
	encoding := flag.Bool("encoding", false, "show instruction encodings")
	rust := flag.String("func", "", "generate rust function with name")
	variant := flag.String("variant", "", "ISA version")

	total := 0

	flag.Parse()
	args := flag.Args()

	if *rust != "" {
		fmt.Print("// AUTO-GENERATED FILE: DO NOT EDIT\n")
		fmt.Printf("pub fn %s(op: Op) -> bool {\n", *rust)
		fmt.Printf("\tmatch op {\n")
	}

	names := make(map[string]bool)
	filepath.WalkDir(args[0], func(path string, insn fs.DirEntry, err error) error {
		if insn != nil && !insn.IsDir() && strings.HasSuffix(path, ".xml") {
			data, err := os.ReadFile(path)
			if err != nil {
				log.Fatal(err)
			}
			var insn InsnSection
			if err := xml.Unmarshal(data, &insn); err != nil {
				return nil
			}

			if insn.Type == "instruction" || insn.Type == "alias" {
				if *base && !insn.BaseVariant() {
					return nil
				}
				hasclass := false
				for _, class := range strings.Split(*classes, ",") {
					if insn.HasClass(class) {
						hasclass = true
					}
				}
				if !hasclass {
					return nil
				}
				if *branch && !insn.IsBranch() {
					return nil
				}
				if *rdmem && !insn.ReadsMem() {
					return nil
				}
				if *wrmem && !insn.WritesMem() {
					return nil
				}
				if *atomic && !insn.MemAtomic() {
					return nil
				}
				if *nomodify && len(insn.WriteSet()) != 0 {
					return nil
				}
				if *variant != "" && !insn.VariantLE(*variant) {
					return nil
				}

				if *rust != "" {
					if filepath.Base(path) == "b_cond.xml" {
						fmt.Print(condbranches)
					} else {
						for _, n := range insn.Names() {
							if !names[n] {
								fmt.Printf("\t\tOp::%s => true,\n", n)
								names[n] = true
							}
						}
					}
				} else {
					fmt.Printf("%s: %s\n", filepath.Base(path), insn.Names())
				}

				total += len(insn.Names())

				if *encoding {
					for _, c := range insn.Classes.IClass {
						fmt.Printf("\t%s: %s\n", c.Id, c.RegDiagram)
					}
				}
			}
		}
		return nil
	})

	if *rust != "" {
		fmt.Printf("\t\t_ => false,\n")
		fmt.Printf("\t}\n}\n\n")
	} else {
		fmt.Printf("total instructions: %d\n", total)
	}
}

var condbranches = `		Op::B_AL => true,
		Op::B_CC => true,
		Op::B_CS => true,
		Op::B_EQ => true,
		Op::B_GE => true,
		Op::B_GT => true,
		Op::B_HI => true,
		Op::B_LE => true,
		Op::B_LS => true,
		Op::B_LT => true,
		Op::B_MI => true,
		Op::B_NE => true,
		Op::B_NV => true,
		Op::B_PL => true,
		Op::B_VC => true,
		Op::B_VS => true,
`
