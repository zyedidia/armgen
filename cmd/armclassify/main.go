package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/praserx/ipconv"
)

type Record struct {
	File       string
	Name       string
	IClass     string
	Path       string
	Variants   string
	Features   string
	InstrClass string
	RegDiagram string
}

func xMask(bits string) string {
	mask := &bytes.Buffer{}
	mask.WriteString("0b")
	for _, c := range bits {
		switch c {
		case 'x':
			mask.WriteByte('0')
		default:
			mask.WriteByte('1')
		}
	}
	return mask.String()
}

func GenerateRegParseExpr(diagram string) string {
	parts := strings.Split(diagram, "|")
	b := 0
	buf := &bytes.Buffer{}
	first := true
	for i := len(parts) - 1; i >= 0; i-- {
		not := false
		_, bits, found := strings.Cut(parts[i], "=")
		if !found {
			_, bits, found = strings.Cut(parts[i], "!=")
			if found {
				not = true
			} else {
				bits = parts[i]
			}
		}

		if strings.Count(bits, "x") != len(bits) {
			mask := xMask(bits)

			var expr string
			if !not {
				expr = fmt.Sprintf("((insn >> %d) & %s) == 0b%s", b, mask, strings.ReplaceAll(bits, "x", "0"))
			} else {
				expr = fmt.Sprintf("((insn >> %d) & %s) != 0b%s", b, mask, bits)
			}
			if first {
				first = false
			} else {
				fmt.Fprint(buf, "&&")
			}
			fmt.Fprintf(buf, "(%s)", expr)
		}

		b += len(bits)
	}
	return buf.String()
}

func GenerateRegParseFunc(i int, diagram string) string {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "func parse_%d(insn uint32) bool {\n", i)
	fmt.Fprintf(buf, "\treturn %s\n", GenerateRegParseExpr(diagram))
	fmt.Fprintf(buf, "}\n")
	return buf.String()
}

func main() {
	gen := flag.Bool("gen", false, "generate parsers")
	out := flag.String("out", "", "output file")
	in := flag.String("in", "", "input file")
	flag.Parse()
	args := flag.Args()

	if *gen {
		if len(args) <= 0 {
			log.Fatal("no input")
		}
		b, err := os.ReadFile(args[0])
		if err != nil {
			log.Fatal(err)
		}
		var records []Record
		err = json.Unmarshal(b, &records)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("package main")
		fmt.Println("var funcs = []func(uint32) bool{")
		for i := range records {
			fmt.Printf("\tparse_%d,\n", i)
		}
		fmt.Println("}")

		for i, r := range records {
			fmt.Println(GenerateRegParseFunc(i, r.RegDiagram))
		}
		return
	}

	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			log.Fatal(err)
		}
		// max := uint64(65536)
		max := uint64(^uint32(0)) + 1
		vals := make([]int16, max)
		n := runtime.NumCPU()
		chunksz := max / uint64(n)
		wg := sync.WaitGroup{}
		for t := 0; t < n; t++ {
			wg.Add(1)
			go func(tid uint64) {
				start := tid * chunksz
				end := tid*chunksz + chunksz
				for i := start; i < end; i++ {
					insn := uint32(i)
					if tid == 0 && i%200000 == 0 {
						fmt.Printf("%.1f\n", float64(i)/float64(end)*100.0)
					}
					found := false
					for j, fn := range funcs {
						if fn(insn) {
							vals[i] = int16(j)
							found = true
							break
						}
					}
					if !found {
						vals[i] = -1
					}
				}
				wg.Done()
			}(uint64(t))
		}
		wg.Wait()
		err = binary.Write(f, binary.LittleEndian, vals)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}

	if *in != "" {
		if len(args) <= 0 {
			log.Fatal("no input")
		}
		jsondat, err := os.ReadFile(args[0])
		if err != nil {
			log.Fatal(err)
		}
		var records []Record
		err = json.Unmarshal(jsondat, &records)
		if err != nil {
			log.Fatal(err)
		}

		f, err := os.Open(*in)
		if err != nil {
			log.Fatal(err)
		}
		b := make([]int16, 4*1024*1024*1024)
		err = binary.Read(f, binary.LittleEndian, b)
		if err != nil {
			log.Fatal(err)
		}
		class := 1
		classes := make(map[string]int)
		for i := 0; i < len(b); i += 256 {
			total := 0
			for j := 0; j < 256; j++ {
				if b[i+j] != -1 && b[i+j] != 0 {
					r := records[b[i+j]]
					if _, ok := classes[r.InstrClass]; !ok {
						classes[r.InstrClass] = class
						class++
					}
					total += classes[r.InstrClass]
				}
			}
			if total != 0 {
				avg := int(float64(total) / float64(256))
				fmt.Printf("%s %d\n", ipconv.IntToIPv4(uint32(i)), avg)
			}
		}
		log.Println(classes)
		log.Println(len(b))
	}

	// iclasses := make(map[string]int)
	// for _, r := range records {
	// 	iclasses[r.IClass]++
	// }
	// fmt.Println("iclasses:", len(iclasses))
	//
	// instrClasses := make(map[string]int)
	// for _, r := range records {
	// 	if r.InstrClass == "" {
	// 		fmt.Println(r.File)
	// 	}
	// 	instrClasses[r.InstrClass]++
	// }
	// fmt.Println("instr-classes:", len(instrClasses), instrClasses)
}
