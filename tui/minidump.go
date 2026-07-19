package tui

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/ianlancetaylor/demangle"
	"github.com/jpnurmi/slope/minidump"
)

func renderMinidump(data []byte, width int) (string, error) {
	md, err := minidump.Parse(data)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	section := func(title string) {
		b.WriteString(labelStyle.Render(title) + "\n")
		w := width
		if w <= 0 {
			w = 40
		}
		b.WriteString(separatorStyle.Render(strings.Repeat("─", w)) + "\n")
	}
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	kv := func(key, value string) {
		b.WriteString(keyStyle.Render(key) + value + "\n")
	}

	if si := md.SystemInfo; si != nil {
		section("System Info")
		os := minidump.OSTypeNames[si.OSType]
		if os == "" {
			os = fmt.Sprintf("OS %d", si.OSType)
		}
		kv("OS:       ", fmt.Sprintf("%s %d.%d Build %d", os, si.OSVerMajor, si.OSVerMinor, si.OSBuild))
		cpu := minidump.CPUArchNames[si.CPUArch]
		if cpu == "" {
			cpu = fmt.Sprintf("arch %d", si.CPUArch)
		}
		kv("CPU:      ", fmt.Sprintf("%s (%d CPUs) Level %d Rev 0x%04X", cpu, si.NumCPUs, si.CPULevel, si.CPURevision))
		if si.ServicePack != "" {
			kv("Service:  ", si.ServicePack)
		}
		b.WriteString("\n")
	}

	if ex := md.Exception; ex != nil {
		section("Exception")
		kv("Thread:   ", fmt.Sprintf("0x%08X", ex.ThreadID))
		code := fmt.Sprintf("0x%08X", ex.Code)
		if name, ok := minidump.ExceptionCodeNames[ex.Code]; ok {
			code += " (" + name + ")"
		}
		kv("Code:     ", code)
		kv("Address:  ", fmt.Sprintf("0x%016X", ex.Address))
		if ex.NumParams > 0 {
			params := make([]string, len(ex.Params))
			for i, p := range ex.Params {
				params[i] = fmt.Sprintf("0x%016X", p)
			}
			kv("Params:   ", "["+strings.Join(params, ", ")+"]")
		}
		b.WriteString("\n")
	}

	if ai := md.AssertionInfo; ai != nil && (ai.Expression != "" || ai.Function != "" || ai.File != "") {
		section("Assertion")
		if ai.Expression != "" {
			kv("Expr:     ", ai.Expression)
		}
		if ai.Function != "" {
			kv("Function: ", ai.Function)
		}
		if ai.File != "" {
			kv("File:     ", fmt.Sprintf("%s:%d", ai.File, ai.Line))
		}
		if name, ok := minidump.AssertionTypeNames[ai.Type]; ok {
			kv("Type:     ", name)
		}
		b.WriteString("\n")
	}

	if len(md.Stacktraces) > 0 {
		totalFrames := 0
		for _, st := range md.Stacktraces {
			totalFrames += len(st.Frames)
		}
		section(fmt.Sprintf("Stacktraces (%d threads, %d frames)", len(md.Stacktraces), totalFrames))
		for i, st := range md.Stacktraces {
			if i > 0 {
				b.WriteString("\n")
			}
			threadLabel := fmt.Sprintf("Thread 0x%08X", st.ThreadID)
			if name, ok := md.ThreadNames[st.ThreadID]; ok {
				threadLabel += fmt.Sprintf(" %q", name)
			}
			if md.Exception != nil && md.Exception.ThreadID == st.ThreadID {
				threadLabel += " (crashed)"
			}
			b.WriteString(threadLabel + "\n")
			for j, f := range st.Frames {
				sym := demangle.Filter(f.Symbol)
				if sym == "" {
					sym = "<unknown>"
				}
				module := ""
				for _, m := range md.Modules {
					if f.InstructionAddr >= m.BaseOfImage && f.InstructionAddr < m.BaseOfImage+uint64(m.SizeOfImage) {
						module = baseName(m.Name)
						break
					}
				}
				line := fmt.Sprintf("  %s %s", keyStyle.Render(fmt.Sprintf("#%-3d 0x%016X", j, f.InstructionAddr)), sym)
				if module != "" {
					line += " " + keyStyle.Render("("+module+")")
				}
				b.WriteString(line + "\n")
			}
		}
		b.WriteString("\n")
	}

	if len(md.Threads) > 0 {
		section(fmt.Sprintf("Threads (%d)", len(md.Threads)))
		for _, t := range md.Threads {
			line := fmt.Sprintf("0x%08X", t.ID)
			line += "  " + keyStyle.Render("Priority:") + fmt.Sprintf(" %d/%d", t.Priority, t.PriorityClass)
			line += "  " + keyStyle.Render("Stack:") + fmt.Sprintf(" 0x%016X (%s)", t.StackStart, formatSize(int(t.StackSize)))
			if name, ok := md.ThreadNames[t.ID]; ok {
				line += "  " + fmt.Sprintf("%q", name)
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if len(md.Modules) > 0 {
		section(fmt.Sprintf("Modules (%d)", len(md.Modules)))
		for _, m := range md.Modules {
			name := baseName(m.Name)
			line := fmt.Sprintf("0x%016X  %-10s  %s", m.BaseOfImage, formatSize(int(m.SizeOfImage)), name)
			if m.VersionMajor != 0 || m.VersionMinor != 0 || m.VersionBuild != 0 {
				line += fmt.Sprintf("  (%d.%d.%d.%d)", m.VersionMajor, m.VersionMinor, m.VersionBuild, m.VersionPatch)
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if len(md.UnloadedModules) > 0 {
		section(fmt.Sprintf("Unloaded Modules (%d)", len(md.UnloadedModules)))
		for _, m := range md.UnloadedModules {
			fmt.Fprintf(&b, "0x%016X  %-10s  %s\n", m.BaseOfImage, formatSize(int(m.SizeOfImage)), baseName(m.Name))
		}
		b.WriteString("\n")
	}

	if mi := md.MiscInfo; mi != nil {
		section("Misc Info")
		kv("PID:      ", fmt.Sprintf("%d", mi.ProcessID))
		if mi.ProcessCreateTime > 0 {
			t := time.Unix(int64(mi.ProcessCreateTime), 0).UTC()
			kv("Created:  ", t.Format("2006-01-02 15:04:05 UTC"))
		}
		if mi.ProcessUserTime > 0 || mi.ProcessKernelTime > 0 {
			kv("User:     ", formatDuration(mi.ProcessUserTime))
			kv("Kernel:   ", formatDuration(mi.ProcessKernelTime))
		}
		b.WriteString("\n")
	}

	if bp := md.BreakpadInfo; bp != nil {
		section("Breakpad Info")
		if bp.Validity&1 != 0 {
			kv("Dump thread:       ", fmt.Sprintf("0x%08X", bp.DumpThreadID))
		}
		if bp.Validity&2 != 0 {
			kv("Requesting thread: ", fmt.Sprintf("0x%08X", bp.RequestingThreadID))
		}
		b.WriteString("\n")
	}

	if sm := md.SystemMemInfo; sm != nil {
		section("System Memory")
		pageSize := uint64(sm.PageSize)
		if pageSize == 0 {
			pageSize = 4096
		}
		kv("Page size:     ", formatSize(int(sm.PageSize)))
		kv("Physical:      ", fmt.Sprintf("%s (%d pages)", formatSize(int(uint64(sm.NumberOfPhysPages)*pageSize)), sm.NumberOfPhysPages))
		kv("Processors:    ", fmt.Sprintf("%d", sm.NumberOfProcessors))
		kv("Available:     ", fmt.Sprintf("%s (%d pages)", formatSize(int(sm.AvailablePages*pageSize)), sm.AvailablePages))
		kv("Committed:     ", fmt.Sprintf("%s (%d pages)", formatSize(int(sm.CommittedPages*pageSize)), sm.CommittedPages))
		kv("Commit limit:  ", fmt.Sprintf("%s (%d pages)", formatSize(int(sm.CommitLimit*pageSize)), sm.CommitLimit))
		kv("Peak commit:   ", fmt.Sprintf("%s (%d pages)", formatSize(int(sm.PeakCommitment*pageSize)), sm.PeakCommitment))
		kv("Paged pool:    ", fmt.Sprintf("%s (%d pages)", formatSize(int(uint64(sm.PagedPoolPages)*pageSize)), sm.PagedPoolPages))
		kv("Nonpaged pool: ", fmt.Sprintf("%s (%d pages)", formatSize(int(uint64(sm.NonPagedPoolPages)*pageSize)), sm.NonPagedPoolPages))
		b.WriteString("\n")
	}

	if vm := md.VmCounters; vm != nil {
		section("Process VM Counters")
		kv("Working set:   ", fmt.Sprintf("%s (peak %s)", formatSize(int(vm.WorkingSetSize)), formatSize(int(vm.PeakWorkingSetSize))))
		kv("Private:       ", formatSize(int(vm.PrivateUsage)))
		kv("Pagefile:      ", fmt.Sprintf("%s (peak %s)", formatSize(int(vm.PagefileUsage)), formatSize(int(vm.PeakPagefileUsage))))
		kv("Page faults:   ", fmt.Sprintf("%d", vm.PageFaultCount))
		if vm.QuotaPagedPoolUsage > 0 || vm.QuotaNonPagedPoolUsage > 0 {
			kv("Paged pool:    ", fmt.Sprintf("%s (peak %s)", formatSize(int(vm.QuotaPagedPoolUsage)), formatSize(int(vm.QuotaPeakPagedPoolUsage))))
			kv("Nonpaged pool: ", fmt.Sprintf("%s (peak %s)", formatSize(int(vm.QuotaNonPagedPoolUsage)), formatSize(int(vm.QuotaPeakNonPagedPoolUsage))))
		}
		b.WriteString("\n")
	}

	if len(md.MemoryRanges) > 0 {
		var total uint64
		for _, r := range md.MemoryRanges {
			total += r.Size
		}
		section(fmt.Sprintf("Memory Regions (%d, %s total)", len(md.MemoryRanges), formatSize(int(total))))
		limit := len(md.MemoryRanges)
		if limit > 100 {
			limit = 100
		}
		for _, r := range md.MemoryRanges[:limit] {
			fmt.Fprintf(&b, "0x%016X  %s\n", r.Address, formatSize(int(r.Size)))
		}
		if len(md.MemoryRanges) > 100 {
			fmt.Fprintf(&b, "... (%d more)\n", len(md.MemoryRanges)-100)
		}
		b.WriteString("\n")
	}

	if len(md.MemoryInfo) > 0 {
		section(fmt.Sprintf("Memory Info (%d regions)", len(md.MemoryInfo)))
		limit := len(md.MemoryInfo)
		if limit > 100 {
			limit = 100
		}
		for _, mi := range md.MemoryInfo[:limit] {
			state := minidump.MemStateNames[mi.State]
			if state == "" {
				state = fmt.Sprintf("0x%X", mi.State)
			}
			prot := minidump.MemProtectNames[mi.Protect]
			if prot == "" {
				prot = fmt.Sprintf("0x%X", mi.Protect)
			}
			typ := minidump.MemTypeNames[mi.Type]
			if typ == "" {
				typ = fmt.Sprintf("0x%X", mi.Type)
			}
			fmt.Fprintf(&b, "0x%016X  %-10s  %-12s  %-18s  %s\n",
				mi.BaseAddress, formatSize(int(mi.RegionSize)), state, prot, typ)
		}
		if len(md.MemoryInfo) > 100 {
			fmt.Fprintf(&b, "... (%d more)\n", len(md.MemoryInfo)-100)
		}
		b.WriteString("\n")
	}

	if len(md.Handles) > 0 {
		section(fmt.Sprintf("Handles (%d)", len(md.Handles)))
		limit := len(md.Handles)
		if limit > 100 {
			limit = 100
		}
		for _, h := range md.Handles[:limit] {
			line := fmt.Sprintf("0x%04X", h.Handle)
			if h.TypeName != "" {
				line += fmt.Sprintf("  %-16s", h.TypeName)
			}
			if h.ObjectName != "" {
				line += "  " + h.ObjectName
			}
			b.WriteString(line + "\n")
		}
		if len(md.Handles) > 100 {
			fmt.Fprintf(&b, "... (%d more)\n", len(md.Handles)-100)
		}
		b.WriteString("\n")
	}

	if len(md.FunctionTables) > 0 {
		total := 0
		for _, ft := range md.FunctionTables {
			total += len(ft.Entries)
		}
		section(fmt.Sprintf("Function Tables (%d tables, %d entries)", len(md.FunctionTables), total))
		for i, ft := range md.FunctionTables {
			if i > 0 {
				b.WriteString("\n")
			}
			// entry: "  0x%08X - 0x%08X" = 26 chars before label
			b.WriteString(fmt.Sprintf("%-26s %s 0x%016X - 0x%016X  (%d entries)\n",
				fmt.Sprintf("0x%016X", ft.BaseAddress),
				keyStyle.Render("Range: "), // 7 chars like "Unwind:"
				ft.MinAddress, ft.MaxAddress, len(ft.Entries)))
			limit := len(ft.Entries)
			if limit > 50 {
				limit = 50
			}
			for _, e := range ft.Entries[:limit] {
				fmt.Fprintf(&b, "  0x%08X - 0x%08X  %s 0x%08X\n",
					e.BeginAddress, e.EndAddress, keyStyle.Render("Unwind:"), e.UnwindInfoAddress)
			}
			if len(ft.Entries) > 50 {
				fmt.Fprintf(&b, "  ... (%d more)\n", len(ft.Entries)-50)
			}
		}
		b.WriteString("\n")
	}

	if md.Comment != "" {
		section("Comment")
		for _, line := range strings.Split(md.Comment, "\n") {
			fmt.Fprintf(&b, "%s\n", line)
		}
		b.WriteString("\n")
	}

	for _, key := range []string{"cpu_info", "proc_status", "lsb_release", "cmd_line", "environ", "maps", "auxv"} {
		if v, ok := md.LinuxStrings[key]; ok {
			section("Linux " + key)
			for _, line := range strings.Split(strings.TrimRight(v, "\n"), "\n") {
				fmt.Fprintf(&b, "%s\n", line)
			}
			b.WriteString("\n")
		}
	}

	for _, us := range md.UnknownStreams {
		name := minidump.StreamTypeNames[us.Type]
		if name == "" {
			name = fmt.Sprintf("Stream %d", us.Type)
		}
		if len(us.Data) == 0 {
			section(fmt.Sprintf("%s (empty)", name))
			b.WriteString("\n")
			continue
		}
		section(fmt.Sprintf("%s (unsupported, %s)", name, formatSize(len(us.Data))))
		for _, line := range strings.Split(strings.TrimRight(hex.Dump(us.Data), "\n"), "\n") {
			fmt.Fprintf(&b, "%s\n", line)
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

func formatDuration(centiseconds uint32) string {
	d := time.Duration(centiseconds) * 10 * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.3fs", d.Seconds())
}

func baseName(path string) string {
	if i := strings.LastIndexAny(path, "\\/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
