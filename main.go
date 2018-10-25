//usr/bin/go run $0 $@ ; exit
package main

import (
	"bytes"
	"debug/elf"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const versionString = "ELFinfo 0.7.1"

var (
	gccMarker   = []byte("GCC: (")
	gnuEnding   = []byte("GNU) ")
	clangMarker = []byte("clang version")
	rustMarker  = []byte("rustc version")
)

// versionSum takes a slice of strings that are the parts of a version number.
// The parts are converted to numbers. If they can't be converted, they count
// as less than nothing. The parts are then summed together, but with more
// emphasis put on the earlier numbers. 2.0.0.0 has emphasis 2000.
// The sum is then returned.
func versionSum(parts []string) int {
	sum := 0
	length := len(parts)
	for i := length - 1; i >= 0; i-- {
		num, err := strconv.Atoi(parts[i])
		if err != nil {
			num = -1
		}
		sum += num * int(math.Pow(float64(10), float64(length-i-1)))
	}
	return sum
}

// firstIsGreater checks if the first version number is greater than the second one.
// It uses a relatively simple algorithm, where all non-numbers counts as less than "0".
func firstIsGreater(a, b string) bool {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	// Expand the shortest version list with zeroes
	for len(aParts) < len(bParts) {
		aParts = append(aParts, "0")
	}
	for len(bParts) < len(aParts) {
		bParts = append(bParts, "0")
	}
	// The two lists that are being compared should be of the same length
	return versionSum(aParts) > versionSum(bParts)
}

// returns the GCC compiler version or an empty string
// example output: "GCC 6.3.1"
// Also handles clang.
func gccver(f *elf.File) string {
	sec := f.Section(".comment")
	if sec == nil {
		return ""
	}
	versionData, errData := sec.Data()
	if errData != nil {
		return ""
	}
	if bytes.Contains(versionData, gccMarker) {
		// Check if this is really clang
		if bytes.Contains(versionData, clangMarker) {
			clangVersionCatcher := regexp.MustCompile(`(\d+\.)(\d+\.)?(\*|\d+)\ `)
			clangVersion := bytes.TrimSpace(clangVersionCatcher.Find(versionData))
			return "Clang " + string(clangVersion)
		}
		// If the bytes are on this form: "GCC: (GNU) 6.3.0GCC: (GNU) 7.2.0",
		// use the largest version number.
		if bytes.Count(versionData, gccMarker) > 1 {
			// Split in to 3 parts, always valid for >=2 instances of gccMarker
			elements := bytes.SplitN(versionData, gccMarker, 3)
			versionA := elements[1]
			versionB := elements[2]
			if bytes.HasPrefix(versionA, gnuEnding) {
				versionA = versionA[5:]
			}
			if bytes.HasPrefix(versionB, gnuEnding) {
				versionB = versionB[5:]
			}
			if firstIsGreater(string(versionA), string(versionB)) {
				versionData = versionA
			} else {
				versionData = versionB
			}
		}
		// Try the first regexp for picking out the version
		versionCatcher1 := regexp.MustCompile(`(\d+\.)(\d+\.)?(\*|\d+)\ `)
		gccVersion := bytes.TrimSpace(versionCatcher1.Find(versionData))
		if len(gccVersion) > 0 {
			return "GCC " + string(gccVersion)
		}
		// Try the second regexp for picking out the version
		versionCatcher2 := regexp.MustCompile(`(\d+\.)(\d+\.)?(\*|\d+)`)
		gccVersion = bytes.TrimSpace(versionCatcher2.Find(versionData))
		if len(gccVersion) > 0 {
			return "GCC " + string(gccVersion)
		}
		return "GCC " + string(gccVersion)[5:]
	}
	return string(versionData)
}

// Returns the Rust compiler version or an empty string
// example output: "Rust 1.27.0"
func rustverUnstripped(f *elf.File) string {
	// Check if there is debug data in the executable, that may contain the version number
	sec := f.Section(".debug_str")
	if sec == nil {
		return ""
	}
	b, errData := sec.Data()
	if errData != nil {
		return ""
	}

	pos1 := bytes.Index(b, rustMarker)
	if pos1 == -1 {
		return ""
	}
	pos1 += len(rustMarker) + 1
	pos2 := bytes.Index(b[pos1:], []byte("("))
	if pos2 == -1 {
		return ""
	}
	pos2 += pos1
	versionString := strings.TrimSpace(string(b[pos1:pos2]))

	return "Rust " + versionString
}

// Returns the Rust compiler version or an empty string,
// from a stripped Rust executable. Does not contain the Rust
// version number.
// Example output: "Rust (GCC 8.1.0)"
func rustverStripped(f *elf.File) string {
	sec := f.Section(".rodata")
	if sec == nil {
		return ""
	}
	b, errData := sec.Data()
	if errData != nil {
		return ""
	}
	foundIndex := bytes.Index(b, []byte("__rust_"))
	if foundIndex <= 0 || b[foundIndex-1] != 0 {
		return ""
	}
	// Rust may use GCC for linking
	if gccVersion := gccver(f); gccVersion != "" {
		return "Rust (" + gccver(f) + ")"
	}
	return "Rust"
}

// returns the Go compiler version or an empty string
// example output: "Go 1.8.3"
func gover(f *elf.File) string {
	sec := f.Section(".rodata")
	if sec == nil {
		return ""
	}
	b, errData := sec.Data()
	if errData != nil {
		return ""
	}
	versionCatcher := regexp.MustCompile(`go(\d+\.)(\d+\.)?(\*|\d+)`)
	goVersion := string(versionCatcher.Find(b))
	if strings.HasPrefix(goVersion, "go") {
		return "Go " + goVersion[2:]
	}
	if goVersion == "" {
		gosec := f.Section(".gosymtab")
		if gosec != nil {
			return "Go (unknown version)"
		}
		return ""
	}
	return goVersion
}

// returns the FPC compiler version or an empty string
// example output: "FPC 3.0.2"
func pasver(f *elf.File) string {
	sec := f.Section(".data")
	if sec == nil {
		return ""
	}
	b, errData := sec.Data()
	if errData != nil {
		return ""
	}
	versionCatcher := regexp.MustCompile(`FPC\ (\d+\.)?(\d+\.)?(\*|\d+)`)
	return string(versionCatcher.Find(b))

}

// TCC has no version number, but it has some signature sections
// Returns "TCC" or an empty string
func tccver(f *elf.File) string {
	// .note.ABI-tag must be missing
	if f.Section(".note.ABI-tag") != nil {
		// TCC does not normally have this section, not TCC
		return ""
	}
	if f.Section(".rodata.cst4") == nil {
		// TCC usually has this section, not TCC
		return ""
	}
	return "TCC"
}

// returns the OCaml compiler version or an empty string
// example output: "OCaml 4.05.0"
func ocamlver(f *elf.File) string {
	sec := f.Section(".rodata")
	if sec == nil {
		return ""
	}
	b, errData := sec.Data()
	if errData != nil {
		return ""
	}
	if !bytes.Contains(b, []byte("[ocaml]")) {
		// Probably not OCaml
		return ""
	}
	versionCatcher := regexp.MustCompile(`(\d+\.)(\d+\.)?(\*|\d+)`)
	ocamlVersion := "OCaml " + string(versionCatcher.Find(b))
	if ocamlVersion == "" {
		return "OCaml (unknown version)"
	}
	return ocamlVersion
}

// compiler takes an *elf.File and tries to find which compiler and version
// it was compiled with, by probing for known locations, strings and patterns.
func compiler(f *elf.File) string {
	// The ordering matters
	if goVersion := gover(f); goVersion != "" {
		return goVersion
	} else if ocamlVersion := ocamlver(f); ocamlVersion != "" {
		return ocamlVersion
	} else if rustVersion := rustverUnstripped(f); rustVersion != "" {
		return rustVersion
	} else if rustVersion := rustverStripped(f); rustVersion != "" {
		return rustVersion
	} else if gccVersion := gccver(f); gccVersion != "" {
		return gccVersion
	} else if pasVersion := pasver(f); pasVersion != "" {
		return pasVersion
	} else if tccVersion := tccver(f); tccVersion != "" {
		return tccVersion
	}
	return "unknown"
}

// stripped returns true if symbols can not be retrieved from the given ELF file
func stripped(f *elf.File) bool {
	_, err := f.Symbols()
	return err != nil
}

func machine2string(m elf.Machine) string {
	// https://golang.org/pkg/debug/elf/
	switch m {
	case elf.EM_M32:
		return "AT&T WE32100"
	case elf.EM_SPARC:
		return "Sun SPARC"
	case elf.EM_386:
		return "Intel i386"
	case elf.EM_68K:
		return "Motorola 68000"
	case elf.EM_88K:
		return "Motorola 88000"
	case elf.EM_860:
		return "Intel i860"
	case elf.EM_MIPS:
		return "MIPS R3000 Big-Endian only"
	case elf.EM_S370:
		return "IBM System/370"
	case elf.EM_MIPS_RS3_LE:
		return "MIPS R3000 Little-Endian"
	case elf.EM_PARISC:
		return "HP PA-RISC"
	case elf.EM_VPP500:
		return "Fujitsu VPP500"
	case elf.EM_SPARC32PLUS:
		return "SPARC v8plus"
	case elf.EM_960:
		return "Intel 80960"
	case elf.EM_PPC:
		return "PowerPC 32-bit"
	case elf.EM_PPC64:
		return "PowerPC 64-bit"
	case elf.EM_S390:
		return "IBM System/390"
	case elf.EM_V800:
		return "NEC V800"
	case elf.EM_FR20:
		return "Fujitsu FR20"
	case elf.EM_RH32:
		return "TRW RH-32"
	case elf.EM_RCE:
		return "Motorola RCE"
	case elf.EM_ARM:
		return "ARM"
	case elf.EM_SH:
		return "Hitachi SH"
	case elf.EM_SPARCV9:
		return "SPARC v9 64-bit"
	case elf.EM_TRICORE:
		return "Siemens TriCore embedded processor"
	case elf.EM_ARC:
		return "Argonaut RISC Core"
	case elf.EM_H8_300:
		return "Hitachi H8/300"
	case elf.EM_H8_300H:
		return "Hitachi H8/300H"
	case elf.EM_H8S:
		return "Hitachi H8S"
	case elf.EM_H8_500:
		return "Hitachi H8/500"
	case elf.EM_IA_64:
		return "Intel IA-64 Processor"
	case elf.EM_MIPS_X:
		return "Stanford MIPS-X"
	case elf.EM_COLDFIRE:
		return "Motorola ColdFire"
	case elf.EM_68HC12:
		return "Motorola M68HC12"
	case elf.EM_MMA:
		return "Fujitsu MMA"
	case elf.EM_PCP:
		return "Siemens PCP"
	case elf.EM_NCPU:
		return "Sony nCPU"
	case elf.EM_NDR1:
		return "Denso NDR1 microprocessor"
	case elf.EM_STARCORE:
		return "Motorola Star*Core processor"
	case elf.EM_ME16:
		return "Toyota ME16 processor"
	case elf.EM_ST100:
		return "STMicroelectronics ST100 processor"
	case elf.EM_TINYJ:
		return "Advanced Logic Corp. TinyJ processor"
	case elf.EM_X86_64:
		return "Advanced Micro Devices x86-64"
	case elf.EM_PDSP:
		return "Sony DSP Processor"
	case elf.EM_PDP10:
		return "Digital Equipment Corp. PDP-10"
	case elf.EM_PDP11:
		return "Digital Equipment Corp. PDP-11"
	case elf.EM_FX66:
		return "Siemens FX66 microcontroller"
	case elf.EM_ST9PLUS:
		return "STMicroelectronics ST9+ 8/16 bit microcontroller"
	case elf.EM_ST7:
		return "STMicroelectronics ST7 8-bit microcontroller"
	case elf.EM_68HC16:
		return "Motorola MC68HC16 Microcontroller"
	case elf.EM_68HC11:
		return "Motorola MC68HC11 Microcontroller"
	case elf.EM_68HC08:
		return "Motorola MC68HC08 Microcontroller"
	case elf.EM_68HC05:
		return "Motorola MC68HC05 Microcontroller"
	case elf.EM_SVX:
		return "Silicon Graphics SVx"
	case elf.EM_ST19:
		return "STMicroelectronics ST19 8-bit microcontroller"
	case elf.EM_VAX:
		return "Digital VAX"
	case elf.EM_CRIS:
		return "Axis Communications 32-bit embedded processor"
	case elf.EM_JAVELIN:
		return "Infineon Technologies 32-bit embedded processor"
	case elf.EM_FIREPATH:
		return "Element 14 64-bit DSP Processor"
	case elf.EM_ZSP:
		return "LSI Logic 16-bit DSP Processor"
	case elf.EM_MMIX:
		return "Donald Knuth's educational 64-bit processor"
	case elf.EM_HUANY:
		return "Harvard University machine-independent object files"
	case elf.EM_PRISM:
		return "SiTera Prism"
	case elf.EM_AVR:
		return "Atmel AVR 8-bit microcontroller"
	case elf.EM_FR30:
		return "Fujitsu FR30"
	case elf.EM_D10V:
		return "Mitsubishi D10V"
	case elf.EM_D30V:
		return "Mitsubishi D30V"
	case elf.EM_V850:
		return "NEC v850"
	case elf.EM_M32R:
		return "Mitsubishi M32R"
	case elf.EM_MN10300:
		return "Matsushita MN10300"
	case elf.EM_MN10200:
		return "Matsushita MN10200"
	case elf.EM_PJ:
		return "picoJava"
	case elf.EM_OPENRISC:
		return "OpenRISC 32-bit embedded processor"
	case elf.EM_ARC_COMPACT:
		return "ARC International ARCompact processor (old spelling/synonym: EM_ARC_A5)"
	case elf.EM_XTENSA:
		return "Tensilica Xtensa Architecture"
	case elf.EM_VIDEOCORE:
		return "Alphamosaic VideoCore processor"
	case elf.EM_TMM_GPP:
		return "Thompson Multimedia General Purpose Processor"
	case elf.EM_NS32K:
		return "National Semiconductor 32000 series"
	case elf.EM_TPC:
		return "Tenor Network TPC processor"
	case elf.EM_SNP1K:
		return "Trebia SNP 1000 processor"
	case elf.EM_ST200:
		return "STMicroelectronics (www.st.com) ST200 microcontroller"
	case elf.EM_IP2K:
		return "Ubicom IP2xxx microcontroller family"
	case elf.EM_MAX:
		return "MAX Processor"
	case elf.EM_CR:
		return "National Semiconductor CompactRISC microprocessor"
	case elf.EM_F2MC16:
		return "Fujitsu F2MC16"
	case elf.EM_MSP430:
		return "Texas Instruments embedded microcontroller msp430"
	case elf.EM_BLACKFIN:
		return "Analog Devices Blackfin (DSP) processor"
	case elf.EM_SE_C33:
		return "S1C33 Family of Seiko Epson processors"
	case elf.EM_SEP:
		return "Sharp embedded microprocessor"
	case elf.EM_ARCA:
		return "Arca RISC Microprocessor"
	case elf.EM_UNICORE:
		return "Microprocessor series from PKU-Unity Ltd. and MPRC of Peking University"
	case elf.EM_EXCESS:
		return "eXcess: 16/32/64-bit configurable embedded CPU"
	case elf.EM_DXP:
		return "Icera Semiconductor Inc. Deep Execution Processor"
	case elf.EM_ALTERA_NIOS2:
		return "Altera Nios II soft-core processor"
	case elf.EM_CRX:
		return "National Semiconductor CompactRISC CRX microprocessor"
	case elf.EM_XGATE:
		return "Motorola XGATE embedded processor"
	case elf.EM_C166:
		return "Infineon C16x/XC16x processor"
	case elf.EM_M16C:
		return "Renesas M16C series microprocessors"
	case elf.EM_DSPIC30F:
		return "Microchip Technology dsPIC30F Digital Signal Controller"
	case elf.EM_CE:
		return "Freescale Communication Engine RISC core"
	case elf.EM_M32C:
		return "Renesas M32C series microprocessors"
	case elf.EM_TSK3000:
		return "Altium TSK3000 core"
	case elf.EM_RS08:
		return "Freescale RS08 embedded processor"
	case elf.EM_SHARC:
		return "Analog Devices SHARC family of 32-bit DSP processors"
	case elf.EM_ECOG2:
		return "Cyan Technology eCOG2 microprocessor"
	case elf.EM_SCORE7:
		return "Sunplus S+core7 RISC processor"
	case elf.EM_DSP24:
		return "New Japan Radio (NJR) 24-bit DSP Processor"
	case elf.EM_VIDEOCORE3:
		return "Broadcom VideoCore III processor"
	case elf.EM_LATTICEMICO32:
		return "RISC processor for Lattice FPGA architecture"
	case elf.EM_SE_C17:
		return "Seiko Epson C17 family"
	case elf.EM_TI_C6000:
		return "The Texas Instruments TMS320C6000 DSP family"
	case elf.EM_TI_C2000:
		return "The Texas Instruments TMS320C2000 DSP family"
	case elf.EM_TI_C5500:
		return "The Texas Instruments TMS320C55x DSP family"
	case elf.EM_TI_ARP32:
		return "Texas Instruments Application Specific RISC Processor, 32bit fetch"
	case elf.EM_TI_PRU:
		return "Texas Instruments Programmable Realtime Unit"
	case elf.EM_MMDSP_PLUS:
		return "STMicroelectronics 64bit VLIW Data Signal Processor"
	case elf.EM_CYPRESS_M8C:
		return "Cypress M8C microprocessor"
	case elf.EM_R32C:
		return "Renesas R32C series microprocessors"
	case elf.EM_TRIMEDIA:
		return "NXP Semiconductors TriMedia architecture family"
	case elf.EM_QDSP6:
		return "QUALCOMM DSP6 Processor"
	case elf.EM_8051:
		return "Intel 8051 and variants"
	case elf.EM_STXP7X:
		return "STMicroelectronics STxP7x family of configurable and extensible RISC processors"
	case elf.EM_NDS32:
		return "Andes Technology compact code size embedded RISC processor family"
	case elf.EM_ECOG1X:
		return "Cyan Technology eCOG1X family"
	case elf.EM_MAXQ30:
		return "Dallas Semiconductor MAXQ30 Core Micro-controllers"
	case elf.EM_XIMO16:
		return "New Japan Radio (NJR) 16-bit DSP Processor"
	case elf.EM_MANIK:
		return "M2000 Reconfigurable RISC Microprocessor"
	case elf.EM_CRAYNV2:
		return "Cray Inc. NV2 vector architecture"
	case elf.EM_RX:
		return "Renesas RX family"
	case elf.EM_METAG:
		return "Imagination Technologies META processor architecture"
	case elf.EM_MCST_ELBRUS:
		return "MCST Elbrus general purpose hardware architecture"
	case elf.EM_ECOG16:
		return "Cyan Technology eCOG16 family"
	case elf.EM_CR16:
		return "National Semiconductor CompactRISC CR16 16-bit microprocessor"
	case elf.EM_ETPU:
		return "Freescale Extended Time Processing Unit"
	case elf.EM_SLE9X:
		return "Infineon Technologies SLE9X core"
	case elf.EM_L10M:
		return "Intel L10M"
	case elf.EM_K10M:
		return "Intel K10M"
	case elf.EM_AARCH64:
		return "ARM 64-bit Architecture (AArch64)"
	case elf.EM_AVR32:
		return "Atmel Corporation 32-bit microprocessor family"
	case elf.EM_STM8:
		return "STMicroeletronics STM8 8-bit microcontroller"
	case elf.EM_TILE64:
		return "Tilera TILE64 multicore architecture family"
	case elf.EM_TILEPRO:
		return "Tilera TILEPro multicore architecture family"
	case elf.EM_MICROBLAZE:
		return "Xilinx MicroBlaze 32-bit RISC soft processor core"
	case elf.EM_CUDA:
		return "NVIDIA CUDA architecture"
	case elf.EM_TILEGX:
		return "Tilera TILE-Gx multicore architecture family"
	case elf.EM_CLOUDSHIELD:
		return "CloudShield architecture family"
	case elf.EM_COREA_1ST:
		return "KIPO-KAIST Core-A 1st generation processor family"
	case elf.EM_COREA_2ND:
		return "KIPO-KAIST Core-A 2nd generation processor family"
	case elf.EM_ARC_COMPACT2:
		return "Synopsys ARCompact V2"
	case elf.EM_OPEN8:
		return "Open8 8-bit RISC soft processor core"
	case elf.EM_RL78:
		return "Renesas RL78 family"
	case elf.EM_VIDEOCORE5:
		return "Broadcom VideoCore V processor"
	case elf.EM_78KOR:
		return "Renesas 78KOR family"
	case elf.EM_56800EX:
		return "Freescale 56800EX Digital Signal Controller (DSC)"
	case elf.EM_BA1:
		return "Beyond BA1 CPU architecture"
	case elf.EM_BA2:
		return "Beyond BA2 CPU architecture"
	case elf.EM_XCORE:
		return "XMOS xCORE processor family"
	case elf.EM_MCHP_PIC:
		return "Microchip 8-bit PIC(r) family"
	case elf.EM_INTEL205:
		return "Reserved by Intel"
	case elf.EM_INTEL206:
		return "Reserved by Intel"
	case elf.EM_INTEL207:
		return "Reserved by Intel"
	case elf.EM_INTEL208:
		return "Reserved by Intel"
	case elf.EM_INTEL209:
		return "Reserved by Intel"
	case elf.EM_KM32:
		return "KM211 KM32 32-bit processor"
	case elf.EM_KMX32:
		return "KM211 KMX32 32-bit processor"
	case elf.EM_KMX16:
		return "KM211 KMX16 16-bit processor"
	case elf.EM_KMX8:
		return "KM211 KMX8 8-bit processor"
	case elf.EM_KVARC:
		return "KM211 KVARC processor"
	case elf.EM_CDP:
		return "Paneve CDP architecture family"
	case elf.EM_COGE:
		return "Cognitive Smart Memory Processor"
	case elf.EM_COOL:
		return "Bluechip Systems CoolEngine"
	case elf.EM_NORC:
		return "Nanoradio Optimized RISC"
	case elf.EM_CSR_KALIMBA:
		return "CSR Kalimba architecture family"
	case elf.EM_Z80:
		return "Zilog Z80"
	case elf.EM_VISIUM:
		return "Controls and Data Services VISIUMcore processor"
	case elf.EM_FT32:
		return "FTDI Chip FT32 high performance 32-bit RISC architecture"
	case elf.EM_MOXIE:
		return "Moxie processor family"
	case elf.EM_AMDGPU:
		return "AMD GPU architecture"
	case elf.EM_RISCV:
		return "RISC-V"
	case elf.EM_LANAI:
		return "Lanai 32-bit processor"
	case elf.EM_BPF:
		return "Linux BPF %G–%@ in-kernel virtual machine"
	case elf.EM_486:
		return "Intel i486"
	case elf.EM_ALPHA_STD:
		return "Digital Alpha (standard value)"
	case elf.EM_ALPHA:
		return "Alpha (written in the absence of an ABI)"
	case elf.EM_NONE:
		fallthrough
	default:
		return "Unknown machine"
	}
}

func examine(filename string, onlyCompilerInfo bool) {
	f, err := elf.Open(filename)
	if err != nil {
		fmt.Printf("%s: %s\n", filename, err.Error())
		os.Exit(1)
	}
	defer f.Close()

	if onlyCompilerInfo {
		fmt.Printf("%v\n", compiler(f))
	} else {
		fmt.Printf("%s: stripped=%v, compiler=%v, byteorder=%v, machine=%v\n", filename, stripped(f), compiler(f), f.ByteOrder, machine2string(f.Machine))
	}
}

func usage() {
	fmt.Println()
	fmt.Println(versionString)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("    elfinfo [OPTION]... [FILE]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("    -c, --compiler          - only detect compiler name and version")
	fmt.Println("    -v, --version           - version info")
	fmt.Println("    -h, --help              - this help output")
	fmt.Println()
}

// Check if the given filename exists.
// If it exists in $PATH, return the full path,
// else return an empty string.
func which(filename string) (string, error) {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		return filename, nil
	}
	for _, directory := range strings.Split(os.Getenv("PATH"), ":") {
		fullPath := path.Join(directory, filename)
		_, err := os.Stat(fullPath)
		if !os.IsNotExist(err) {
			return fullPath, nil
		}
	}
	return "", errors.New(filename + ": no such file or directory")
}

func main() {
	if len(os.Args) == 2 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			usage()
		} else if os.Args[1] == "-v" || os.Args[1] == "--version" {
			fmt.Println(versionString)
		} else {
			filepath, err := which(os.Args[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			examine(filepath, false)
		}
	} else if len(os.Args) == 3 {
		if os.Args[1] == "-c" || os.Args[1] == "--compiler" {
			filepath, err := which(os.Args[2])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			examine(filepath, true)
		} else {
			usage()
		}
	} else {
		usage()
	}
}
