package tui

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"

	tea "charm.land/bubbletea/v2"
	"github.com/getsentry/slope/envelope"
)

// buildTestMinidump constructs a minimal valid minidump binary for TUI tests.
func buildTestMinidump() []byte {
	le := binary.LittleEndian
	buf := make([]byte, 32) // header

	le.PutUint32(buf[0:], 0x504D444D) // MDMP
	le.PutUint16(buf[4:], 0xA793)
	le.PutUint32(buf[20:], 1700000000)

	writeStr := func(s string) uint32 {
		runes := utf16.Encode([]rune(s))
		rva := uint32(len(buf))
		b := make([]byte, 4+len(runes)*2)
		le.PutUint32(b[0:], uint32(len(runes)*2))
		for i, r := range runes {
			le.PutUint16(b[4+i*2:], r)
		}
		buf = append(buf, b...)
		return rva
	}

	writeUTF16Fixed := func(s string, size int) {
		data := make([]byte, size)
		runes := utf16.Encode([]rune(s))
		for i, r := range runes {
			if i*2+1 >= size {
				break
			}
			le.PutUint16(data[i*2:], r)
		}
		buf = append(buf, data...)
	}

	addStream := func(typ, size, rva uint32) {
		e := make([]byte, 12)
		le.PutUint32(e[0:], typ)
		le.PutUint32(e[4:], size)
		le.PutUint32(e[8:], rva)
		buf = append(buf, e...)
	}

	spRVA := writeStr("Service Pack 1")
	modRVA := writeStr("C:\\test.exe")
	threadNameRVA := writeStr("worker")
	unloadedRVA := writeStr("old.dll")
	handleTypeRVA := writeStr("File")
	handleObjRVA := writeStr("\\Device\\HarddiskVolume")

	// SystemInfo (56 bytes)
	siRVA := uint32(len(buf))
	si := make([]byte, 56)
	le.PutUint16(si[0:], 9) // AMD64
	si[6] = 4
	le.PutUint32(si[12:], 10)
	le.PutUint32(si[20:], 19041)
	le.PutUint32(si[28:], spRVA)
	buf = append(buf, si...)

	// ExceptionStream (168 bytes)
	exRVA := uint32(len(buf))
	ex := make([]byte, 168)
	le.PutUint32(ex[0:], 0x1A2B)
	le.PutUint32(ex[8:], 0xC0000005)
	le.PutUint64(ex[24:], 0x7FF6A1B23456)
	le.PutUint32(ex[32:], 2)
	le.PutUint64(ex[40:], 0)
	le.PutUint64(ex[48:], 0x7FF6A1B23456)
	buf = append(buf, ex...)

	// ThreadList: 1 thread (52 bytes)
	tlRVA := uint32(len(buf))
	tl := make([]byte, 52)
	le.PutUint32(tl[0:], 1)
	le.PutUint32(tl[4:], 0x1A2B)
	le.PutUint32(tl[12:], 256)
	le.PutUint32(tl[16:], 8)
	le.PutUint32(tl[36:], 0x2000)
	buf = append(buf, tl...)

	// ThreadNames: 1 entry (16 bytes)
	tnRVA := uint32(len(buf))
	tn := make([]byte, 16)
	le.PutUint32(tn[0:], 1)
	le.PutUint32(tn[4:], 0x1A2B)
	le.PutUint64(tn[8:], uint64(threadNameRVA))
	buf = append(buf, tn...)

	// ModuleList: 1 module (112 bytes)
	mlRVA := uint32(len(buf))
	ml := make([]byte, 112)
	le.PutUint32(ml[0:], 1)
	le.PutUint32(ml[12:], 0x26000)
	le.PutUint32(ml[28:], modRVA)
	le.PutUint32(ml[32:], (1<<16)|0)
	le.PutUint32(ml[36:], (2<<16)|0)
	buf = append(buf, ml...)

	// UnloadedModuleList: header(12) + 1 entry(24) = 36 bytes
	umRVA := uint32(len(buf))
	um := make([]byte, 36)
	le.PutUint32(um[0:], 12)
	le.PutUint32(um[4:], 24)
	le.PutUint32(um[8:], 1)
	le.PutUint32(um[20:], 0x10000)
	le.PutUint32(um[32:], unloadedRVA)
	buf = append(buf, um...)

	// HandleData: header(16) + 1 entry(32) = 48 bytes
	hdRVA := uint32(len(buf))
	hd := make([]byte, 48)
	le.PutUint32(hd[0:], 16)
	le.PutUint32(hd[4:], 32)
	le.PutUint32(hd[8:], 1)
	le.PutUint64(hd[16:], 0x0ABC)
	le.PutUint32(hd[24:], handleTypeRVA)
	le.PutUint32(hd[28:], handleObjRVA)
	buf = append(buf, hd...)

	// MiscInfo (24 bytes)
	miRVA := uint32(len(buf))
	mi := make([]byte, 24)
	le.PutUint32(mi[0:], 24)
	le.PutUint32(mi[8:], 1234)
	le.PutUint32(mi[12:], 1700000000)
	le.PutUint32(mi[16:], 234)
	le.PutUint32(mi[20:], 156)
	buf = append(buf, mi...)

	// MemoryInfoList: header(16) + 1 entry(48) = 64 bytes
	miLRVA := uint32(len(buf))
	miL := make([]byte, 64)
	le.PutUint32(miL[0:], 16)
	le.PutUint32(miL[4:], 48)
	le.PutUint64(miL[8:], 1)
	le.PutUint32(miL[48:], 0x1000)    // State = MEM_COMMIT
	le.PutUint32(miL[52:], 0x02)      // Protect = READONLY
	le.PutUint32(miL[56:], 0x1000000) // Type = MEM_IMAGE
	buf = append(buf, miL...)

	// SystemMemoryInfo (492 bytes)
	smRVA := uint32(len(buf))
	sm := make([]byte, 0x1EC)
	le.PutUint16(sm[0:], 1)
	le.PutUint32(sm[0x04+0x04:], 4096)
	le.PutUint32(sm[0x04+0x08:], 2097152)
	le.PutUint32(sm[0x04+0x30:], 4)
	le.PutUint64(sm[0x74:], 500000)
	le.PutUint64(sm[0x74+0x08:], 1000000)
	le.PutUint64(sm[0x74+0x10:], 2500000)
	le.PutUint64(sm[0x74+0x18:], 1500000)
	le.PutUint32(sm[0x94+0x70:], 30000)
	le.PutUint32(sm[0x94+0x74:], 15000)
	buf = append(buf, sm...)

	// FunctionTable: header(24) + descriptor(32) + 1 entry(12) = 68 bytes
	ftRVA := uint32(len(buf))
	ft := make([]byte, 68)
	le.PutUint32(ft[0:], 24)
	le.PutUint32(ft[4:], 32)
	le.PutUint32(ft[12:], 12)
	le.PutUint32(ft[16:], 1)
	le.PutUint64(ft[24:], 0x1000)
	le.PutUint64(ft[32:], 0x2000)
	le.PutUint64(ft[40:], 0x00400000)
	le.PutUint32(ft[48:], 1)
	le.PutUint32(ft[56:], 0x1000)
	le.PutUint32(ft[60:], 0x1100)
	le.PutUint32(ft[64:], 0x1200)
	buf = append(buf, ft...)

	// ProcessVmCounters (80 bytes)
	vmRVA := uint32(len(buf))
	vm := make([]byte, 0x50)
	le.PutUint16(vm[0:], 1)
	le.PutUint32(vm[0x04:], 5000)
	le.PutUint64(vm[0x08:], 50*1024*1024)
	le.PutUint64(vm[0x10:], 30*1024*1024)
	le.PutUint64(vm[0x18:], 2*1024*1024)  // QuotaPeakPagedPoolUsage
	le.PutUint64(vm[0x20:], 1*1024*1024)  // QuotaPagedPoolUsage
	le.PutUint64(vm[0x28:], 512*1024)     // QuotaPeakNonPagedPoolUsage
	le.PutUint64(vm[0x30:], 256*1024)     // QuotaNonPagedPoolUsage
	le.PutUint64(vm[0x38:], 20*1024*1024)
	le.PutUint64(vm[0x40:], 40*1024*1024)
	le.PutUint64(vm[0x48:], 25*1024*1024)
	buf = append(buf, vm...)

	// BreakpadInfo (12 bytes)
	bpRVA := uint32(len(buf))
	bp := make([]byte, 12)
	le.PutUint32(bp[0:], 0x03)
	le.PutUint32(bp[4:], 0x1A2B)
	le.PutUint32(bp[8:], 0x3C4D)
	buf = append(buf, bp...)

	// AssertionInfo (776 bytes)
	aiRVA := uint32(len(buf))
	writeUTF16Fixed("x != 0", 256)
	writeUTF16Fixed("init", 256)
	writeUTF16Fixed("main.c", 256)
	aiLine := len(buf)
	buf = append(buf, make([]byte, 8)...)
	le.PutUint32(buf[aiLine:], 99)
	le.PutUint32(buf[aiLine+4:], 1)

	// CommentA
	comment := "dump comment"
	caRVA := uint32(len(buf))
	buf = append(buf, []byte(comment)...)

	// Linux lsb_release
	lsb := "Ubuntu 22.04"
	lsbRVA := uint32(len(buf))
	buf = append(buf, []byte(lsb)...)

	// SentryStackTraces: header(16) + 1 thread(12) + 2 frames(32) + symbols
	mangledSym := "_ZN5MyApp13trigger_crashEv"
	plainSym := "plain_func"
	symbolData := mangledSym + plainSym
	stRVA := uint32(len(buf))
	st := make([]byte, 16+12+4+32) // header(16) + 1 thread(12) + align(4) + 2 frames(32)
	le.PutUint32(st[0:], 1)                           // version
	le.PutUint32(st[4:], 1)                            // num_threads
	le.PutUint32(st[8:], 2)                            // num_frames
	le.PutUint32(st[12:], uint32(len(symbolData)))     // symbol_bytes
	le.PutUint32(st[16:], 0x1A2B)                      // thread_id
	le.PutUint32(st[20:], 0)                           // start_frame
	le.PutUint32(st[24:], 2)                           // num_frames
	// 4 bytes padding at [28:32] for 8-byte alignment
	le.PutUint64(st[32:], 0x7FF6A1B23456)              // frame 0 addr
	le.PutUint32(st[40:], 0)                           // symbol_offset
	le.PutUint32(st[44:], uint32(len(mangledSym)))     // symbol_len
	le.PutUint64(st[48:], 0x7FF6A1B24000)              // frame 1 addr
	le.PutUint32(st[56:], uint32(len(mangledSym)))     // symbol_offset
	le.PutUint32(st[60:], uint32(len(plainSym)))       // symbol_len
	buf = append(buf, st...)
	buf = append(buf, []byte(symbolData)...)
	stSize := uint32(len(buf)) - stRVA

	// Unsupported known stream (ThreadExList = type 8)
	unsupData := []byte{0x01, 0x02, 0x03, 0x04}
	unsupRVA := uint32(len(buf))
	buf = append(buf, unsupData...)

	// Truly unknown stream (type 999)
	unknownData := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	unkRVA := uint32(len(buf))
	buf = append(buf, unknownData...)

	// Stream directory
	dirRVA := uint32(len(buf))
	type stream struct {
		typ, size, rva uint32
	}
	streams := []stream{
		{7, 56, siRVA},
		{6, 168, exRVA},
		{3, 52, tlRVA},
		{24, 16, tnRVA},
		{4, 112, mlRVA},
		{14, 36, umRVA},
		{12, 48, hdRVA},
		{15, 24, miRVA},
		{16, 64, miLRVA},
		{13, 68, ftRVA},
		{21, 0x1EC, smRVA},
		{22, 0x50, vmRVA},
		{0x47670001, 12, bpRVA},
		{0x47670002, 776, aiRVA},
		{10, uint32(len(comment)), caRVA},
		{0x47670005, uint32(len(lsb)), lsbRVA},
		{0x53790001, stSize, stRVA},
		{8, uint32(len(unsupData)), unsupRVA},
		{999, uint32(len(unknownData)), unkRVA},
	}
	for _, s := range streams {
		addStream(s.typ, s.size, s.rva)
	}

	le.PutUint32(buf[8:], uint32(len(streams)))
	le.PutUint32(buf[12:], dirRVA)

	return buf
}

func TestRenderMinidump(t *testing.T) {
	data := buildTestMinidump()
	got, err := renderMinidump(data, 80)
	if err != nil {
		t.Fatalf("renderMinidump: %v", err)
	}

	for _, want := range []string{
		"System Info",
		"Windows NT 10.0 Build 19041",
		"AMD64 (4 CPUs)",
		"Service Pack 1",
		"Exception",
		"0x00001A2B",
		"0xC0000005 (ACCESS_VIOLATION)",
		"0x00007FF6A1B23456",
		"Params:",
		"Assertion",
		"x != 0",
		"init",
		"main.c:99",
		"INVALID_PARAMETER",
		"Stacktraces (1 threads, 2 frames)",
		"MyApp::trigger_crash()",
		"plain_func",
		"(crashed)",
		"Threads (1)",
		`"worker"`,
		"Modules (1)",
		"test.exe",
		"(1.0.2.0)",
		"Unloaded Modules (1)",
		"old.dll",
		"Handles (1)",
		"0x0ABC",
		"File",
		"\\Device\\HarddiskVolume",
		"Misc Info",
		"PID:",
		"1234",
		"Created:",
		"User:",
		"Kernel:",
		"Function Tables",
		"0x00001000 - 0x00001100",
		"Unwind:",
		"0x00001200",
		"Process VM Counters",
		"Working set:",
		"Private:",
		"Pagefile:",
		"Page faults:",
		"System Memory",
		"Page size:",
		"Physical:",
		"Available:",
		"Committed:",
		"Paged pool:",
		"Nonpaged pool:",
		"Breakpad Info",
		"Dump thread:",
		"Requesting thread:",
		"Memory Info",
		"MEM_COMMIT",
		"READONLY",
		"MEM_IMAGE",
		"Comment",
		"dump comment",
		"Linux lsb_release",
		"Ubuntu 22.04",
		"ThreadExList (unsupported",
		"Stream 999 (unsupported",
		"de ad be ef",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestRenderMinidumpError(t *testing.T) {
	_, err := renderMinidump([]byte("not a minidump"), 80)
	if err == nil {
		t.Error("expected error for invalid minidump data")
	}
}

func TestPagerContentMinidump(t *testing.T) {
	data := buildTestMinidump()
	m := testModel(0)
	m.envelope.Items = []envelope.Item{{
		Header:  json.RawMessage(`{"type":"attachment"}`),
		Payload: data,
		Type:    "attachment",
	}}
	got := m.pagerContent()
	if !strings.Contains(got, "System Info") {
		t.Error("pagerContent for minidump should contain System Info")
	}
	if !strings.Contains(got, "Exception") {
		t.Error("pagerContent for minidump should contain Exception")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		cs   uint32
		want string
	}{
		{0, "0ms"},
		{5, "50ms"},
		{100, "1.000s"},
		{234, "2.340s"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.cs)
		if got != tt.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.cs, got, tt.want)
		}
	}
}

func TestRenderMinidumpLinuxStrings(t *testing.T) {
	le := binary.LittleEndian
	buf := make([]byte, 32)
	le.PutUint32(buf[0:], 0x504D444D)

	cpuInfo := "model name\t: Intel\n"
	cpuRVA := uint32(len(buf))
	buf = append(buf, []byte(cpuInfo)...)

	dirRVA := uint32(len(buf))
	e := make([]byte, 12)
	le.PutUint32(e[0:], 0x47670003)
	le.PutUint32(e[4:], uint32(len(cpuInfo)))
	le.PutUint32(e[8:], cpuRVA)
	buf = append(buf, e...)

	le.PutUint32(buf[8:], 1)
	le.PutUint32(buf[12:], dirRVA)

	got, err := renderMinidump(buf, 80)
	if err != nil {
		t.Fatalf("renderMinidump: %v", err)
	}
	if !strings.Contains(got, "Linux cpu_info") {
		t.Error("output missing Linux cpu_info section")
	}
	if !strings.Contains(got, "Intel") {
		t.Error("output missing cpu info content")
	}
}

func TestRenderMinidumpUnknownCode(t *testing.T) {
	le := binary.LittleEndian
	buf := make([]byte, 32)
	le.PutUint32(buf[0:], 0x504D444D)

	exRVA := uint32(len(buf))
	ex := make([]byte, 168)
	le.PutUint32(ex[0:], 0x1A2B)
	le.PutUint32(ex[8:], 0xDEAD)
	buf = append(buf, ex...)

	dirRVA := uint32(len(buf))
	e := make([]byte, 12)
	le.PutUint32(e[0:], 6)
	le.PutUint32(e[4:], 168)
	le.PutUint32(e[8:], exRVA)
	buf = append(buf, e...)

	le.PutUint32(buf[8:], 1)
	le.PutUint32(buf[12:], dirRVA)

	got, err := renderMinidump(buf, 80)
	if err != nil {
		t.Fatalf("renderMinidump: %v", err)
	}
	if !strings.Contains(got, "0x0000DEAD") {
		t.Error("output missing hex exception code")
	}
	if strings.Contains(got, "(") {
		t.Error("unknown code should not have a name in parentheses")
	}
}

func TestRenderMinidumpTruncation(t *testing.T) {
	le := binary.LittleEndian

	// Build a minidump with 101 memory regions to trigger the >100 truncation
	buf := make([]byte, 32)
	le.PutUint32(buf[0:], 0x504D444D)
	le.PutUint16(buf[4:], 0xA793)

	// MemoryList: 4 + 101*16 = 1620 bytes
	memRVA := uint32(len(buf))
	mem := make([]byte, 4+101*16)
	le.PutUint32(mem[0:], 101)
	for i := 0; i < 101; i++ {
		off := 4 + i*16
		le.PutUint64(mem[off:], uint64(i*0x1000))
		le.PutUint32(mem[off+8:], 0x1000)
	}
	buf = append(buf, mem...)

	// MemoryInfoList: 16 + 101*48 = 4864 bytes
	miRVA := uint32(len(buf))
	mi := make([]byte, 16+101*48)
	le.PutUint32(mi[0:], 16)
	le.PutUint32(mi[4:], 48)
	le.PutUint64(mi[8:], 101)
	for i := 0; i < 101; i++ {
		off := 16 + i*48
		le.PutUint64(mi[off:], uint64(i*0x1000))
		le.PutUint64(mi[off+24:], 0x1000)
		le.PutUint32(mi[off+32:], 0x1000) // MEM_COMMIT
		le.PutUint32(mi[off+36:], 0x04)   // READWRITE
		le.PutUint32(mi[off+40:], 0x20000) // MEM_PRIVATE
	}
	buf = append(buf, mi...)

	dirRVA := uint32(len(buf))
	for _, s := range [][3]uint32{
		{5, uint32(len(mem)), memRVA},
		{16, uint32(len(mi)), miRVA},
	} {
		e := make([]byte, 12)
		le.PutUint32(e[0:], s[0])
		le.PutUint32(e[4:], s[1])
		le.PutUint32(e[8:], s[2])
		buf = append(buf, e...)
	}
	le.PutUint32(buf[8:], 2)
	le.PutUint32(buf[12:], dirRVA)

	got, err := renderMinidump(buf, 80)
	if err != nil {
		t.Fatalf("renderMinidump: %v", err)
	}
	if !strings.Contains(got, "... (1 more)") {
		t.Error("output should contain truncation indicator")
	}
	if !strings.Contains(got, "Memory Regions (101") {
		t.Error("output should show 101 memory regions")
	}
	if !strings.Contains(got, "Memory Info (101") {
		t.Error("output should show 101 memory info entries")
	}
}

func TestPagerContentMinidumpError(t *testing.T) {
	payload := []byte("MDMP\x00\x00\x00\x00")
	m := testModel(0)
	m.envelope.Items = []envelope.Item{{
		Header:  json.RawMessage(`{"type":"attachment"}`),
		Payload: payload,
		Type:    "attachment",
	}}
	got := m.pagerContent()
	want := hex.Dump(payload)
	if got != want {
		t.Errorf("pagerContent for invalid minidump should return hex dump")
	}
}

func TestNewMinidumpViewer(t *testing.T) {
	data := buildTestMinidump()
	m, err := NewMinidumpViewer(data, "crash.dmp", int64(len(data)))
	if err != nil {
		t.Fatalf("NewMinidumpViewer: %v", err)
	}
	if m.mode != modeMinidump {
		t.Errorf("mode = %d, want modeMinidump (%d)", m.mode, modeMinidump)
	}
	if m.filePath != "crash.dmp" {
		t.Errorf("filePath = %q, want %q", m.filePath, "crash.dmp")
	}
	if m.fileSize != int64(len(data)) {
		t.Errorf("fileSize = %d, want %d", m.fileSize, len(data))
	}
}

func TestNewMinidumpViewerInvalidData(t *testing.T) {
	_, err := NewMinidumpViewer([]byte{0, 1, 2}, "invalid.dmp", 3)
	if err == nil {
		t.Fatal("expected error for invalid minidump data")
	}
}

func TestMinidumpViewerQuit(t *testing.T) {
	data := buildTestMinidump()
	m, err := NewMinidumpViewer(data, "crash.dmp", int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	_, cmd := m.Update(key('q'))
	if !isQuitCmd(cmd) {
		t.Error("q in minidump mode: expected quit cmd")
	}

	m, _ = NewMinidumpViewer(data, "crash.dmp", int64(len(data)))
	_, cmd = m.Update(specialKey(tea.KeyEscape))
	if !isQuitCmd(cmd) {
		t.Error("esc in minidump mode: expected quit cmd")
	}
}

func TestMinidumpViewerScroll(t *testing.T) {
	data := buildTestMinidump()
	m, err := NewMinidumpViewer(data, "crash.dmp", int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	m.pager.SetWidth(80)
	m.pager.SetHeight(5)

	next, _ := m.Update(key('j'))
	m = next.(Model)
	if m.mode != modeMinidump {
		t.Errorf("j in minidump: mode = %d, want modeMinidump", m.mode)
	}
}
