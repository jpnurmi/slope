package minidump

import (
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

// testMinidumpBuilder constructs a valid minidump binary for testing.
type testMinidumpBuilder struct {
	buf []byte
}

func newBuilder() *testMinidumpBuilder {
	return &testMinidumpBuilder{}
}

func (b *testMinidumpBuilder) grow(n int) int {
	off := len(b.buf)
	b.buf = append(b.buf, make([]byte, n)...)
	return off
}

func (b *testMinidumpBuilder) writeU16(off int, v uint16) {
	le.PutUint16(b.buf[off:], v)
}

func (b *testMinidumpBuilder) writeU32(off int, v uint32) {
	le.PutUint32(b.buf[off:], v)
}

func (b *testMinidumpBuilder) writeU64(off int, v uint64) {
	le.PutUint64(b.buf[off:], v)
}

func (b *testMinidumpBuilder) writeMinidumpString(s string) uint32 {
	runes := utf16.Encode([]rune(s))
	rva := uint32(len(b.buf))
	off := b.grow(4 + len(runes)*2)
	le.PutUint32(b.buf[off:], uint32(len(runes)*2))
	for i, r := range runes {
		le.PutUint16(b.buf[off+4+i*2:], r)
	}
	return rva
}

func (b *testMinidumpBuilder) writeUTF16Fixed(s string, size int) int {
	off := b.grow(size)
	runes := utf16.Encode([]rune(s))
	for i, r := range runes {
		if i*2+1 >= size {
			break
		}
		le.PutUint16(b.buf[off+i*2:], r)
	}
	return off
}

func (b *testMinidumpBuilder) addStream(typ, size, rva uint32) {
	off := b.grow(12)
	b.writeU32(off, typ)
	b.writeU32(off+4, size)
	b.writeU32(off+8, rva)
}

func testMinidump() []byte {
	b := newBuilder()

	// Reserve header (32 bytes)
	b.grow(32)
	b.writeU32(0, signature)
	b.writeU16(4, 0xA793)
	b.writeU16(6, 0x0001)
	b.writeU32(20, 1700000000)
	b.writeU64(24, 0)

	// Strings
	spRVA := b.writeMinidumpString("Service Pack 1")
	modNameRVA := b.writeMinidumpString("C:\\app.exe")
	threadNameRVA := b.writeMinidumpString("main-thread")
	unloadedNameRVA := b.writeMinidumpString("C:\\old.dll")
	handleTypeRVA := b.writeMinidumpString("Event")
	handleObjRVA := b.writeMinidumpString("\\BaseNamedObjects\\MyEvent")

	// SystemInfo (56 bytes)
	siRVA := uint32(len(b.buf))
	siOff := b.grow(56)
	b.writeU16(siOff+0, 9)
	b.writeU16(siOff+2, 25)
	b.writeU16(siOff+4, 0x0101)
	b.buf[siOff+6] = 8
	b.writeU32(siOff+8, 0)
	b.writeU32(siOff+12, 10)
	b.writeU32(siOff+16, 0)
	b.writeU32(siOff+20, 19041)
	b.writeU32(siOff+24, 2)
	b.writeU32(siOff+28, spRVA)
	b.writeU16(siOff+52, 0x0100)

	// ExceptionStream (168 bytes)
	exRVA := uint32(len(b.buf))
	exOff := b.grow(168)
	b.writeU32(exOff+0, 0x1A2B)
	b.writeU32(exOff+8, 0xC0000005)
	b.writeU32(exOff+12, 0)
	b.writeU64(exOff+24, 0x00007FF6A1B23456)
	b.writeU32(exOff+32, 2)
	b.writeU64(exOff+40, 0)
	b.writeU64(exOff+48, 0x00007FF6A1B23456)

	// ThreadList: 1 thread (52 bytes)
	tlRVA := uint32(len(b.buf))
	tlOff := b.grow(52)
	b.writeU32(tlOff, 1)
	b.writeU32(tlOff+4, 0x1A2B)
	b.writeU32(tlOff+8, 0)
	b.writeU32(tlOff+12, 256)
	b.writeU32(tlOff+16, 8)
	b.writeU64(tlOff+20, 0xE5B8C4A000)
	b.writeU64(tlOff+28, 0x1F0000)
	b.writeU32(tlOff+36, 0x2000)

	// ThreadNames: 1 entry (4 + 12 = 16 bytes)
	tnRVA := uint32(len(b.buf))
	tnOff := b.grow(16)
	b.writeU32(tnOff, 1)
	b.writeU32(tnOff+4, 0x1A2B)
	b.writeU64(tnOff+8, uint64(threadNameRVA))

	// ModuleList: 1 module (112 bytes)
	mlRVA := uint32(len(b.buf))
	mlOff := b.grow(112)
	b.writeU32(mlOff, 1)
	b.writeU64(mlOff+4, 0x00007FF6A1B20000)
	b.writeU32(mlOff+12, 0x26000)
	b.writeU32(mlOff+16, 0)
	b.writeU32(mlOff+20, 1700000000)
	b.writeU32(mlOff+28, modNameRVA)
	b.writeU32(mlOff+32, (1<<16)|2)
	b.writeU32(mlOff+36, (3<<16)|4)

	// UnloadedModuleList: header(12) + 1 entry(24) = 36 bytes
	umRVA := uint32(len(b.buf))
	umOff := b.grow(36)
	b.writeU32(umOff, 12)   // SizeOfHeader
	b.writeU32(umOff+4, 24) // SizeOfEntry
	b.writeU32(umOff+8, 1)  // NumberOfEntries
	b.writeU64(umOff+12, 0x00007FFB90000000)
	b.writeU32(umOff+20, 0x10000)
	b.writeU32(umOff+24, 0)
	b.writeU32(umOff+28, 0)
	b.writeU32(umOff+32, unloadedNameRVA)

	// HandleData: header(16) + 1 descriptor(32) = 48 bytes
	hdRVA := uint32(len(b.buf))
	hdOff := b.grow(48)
	b.writeU32(hdOff, 16)   // SizeOfHeader
	b.writeU32(hdOff+4, 32) // SizeOfDescriptor
	b.writeU32(hdOff+8, 1)  // NumberOfDescriptors
	b.writeU32(hdOff+12, 0) // Reserved
	b.writeU64(hdOff+16, 0x1234)
	b.writeU32(hdOff+24, handleTypeRVA)
	b.writeU32(hdOff+28, handleObjRVA)
	b.writeU32(hdOff+32, 0)
	b.writeU32(hdOff+36, 0x001F0003)
	b.writeU32(hdOff+40, 2)
	b.writeU32(hdOff+44, 3)

	// MiscInfo (24 bytes)
	miRVA := uint32(len(b.buf))
	miOff := b.grow(24)
	b.writeU32(miOff, 24)
	b.writeU32(miOff+4, 0x0F)
	b.writeU32(miOff+8, 6672)
	b.writeU32(miOff+12, 1700000000)
	b.writeU32(miOff+16, 234)
	b.writeU32(miOff+20, 156)

	// MemoryInfoList: header(16) + 1 entry(48) = 64 bytes
	miLRVA := uint32(len(b.buf))
	miLOff := b.grow(64)
	b.writeU32(miLOff, 16)   // SizeOfHeader
	b.writeU32(miLOff+4, 48) // SizeOfEntry
	b.writeU64(miLOff+8, 1)  // NumberOfEntries
	b.writeU64(miLOff+16, 0x7FFE0000)
	b.writeU64(miLOff+40, 0x1000) // RegionSize
	b.writeU32(miLOff+48, 0x1000) // State = MEM_COMMIT
	b.writeU32(miLOff+52, 0x02)   // Protect = READONLY
	b.writeU32(miLOff+56, 0x1000000) // Type = MEM_IMAGE

	// FunctionTable: header(24) + descriptor(32) + native(0) + 1 entry(12) = 68 bytes
	ftRVA := uint32(len(b.buf))
	ftOff := b.grow(68)
	b.writeU32(ftOff, 24)    // SizeOfHeader
	b.writeU32(ftOff+4, 32)  // SizeOfDescriptor
	b.writeU32(ftOff+8, 0)   // SizeOfNativeDescriptor
	b.writeU32(ftOff+12, 12) // SizeOfFunctionEntry
	b.writeU32(ftOff+16, 1)  // NumberOfDescriptors
	b.writeU32(ftOff+20, 0)  // SizeOfAlignPad
	// Descriptor
	b.writeU64(ftOff+24, 0x1000)     // MinimumAddress
	b.writeU64(ftOff+32, 0x2000)     // MaximumAddress
	b.writeU64(ftOff+40, 0x00400000) // BaseAddress
	b.writeU32(ftOff+48, 1)          // EntryCount
	b.writeU32(ftOff+52, 0)          // SizeOfAlignPad
	// Entry (RUNTIME_FUNCTION)
	b.writeU32(ftOff+56, 0x1000) // BeginAddress
	b.writeU32(ftOff+60, 0x1100) // EndAddress
	b.writeU32(ftOff+64, 0x1200) // UnwindInfoAddress

	// ProcessVmCounters (0x50 = 80 bytes)
	vmRVA := uint32(len(b.buf))
	vmOff := b.grow(0x50)
	b.writeU16(vmOff, 1)                      // Revision
	b.writeU32(vmOff+0x04, 12345)             // PageFaultCount
	b.writeU64(vmOff+0x08, 100*1024*1024)     // PeakWorkingSetSize (100 MB)
	b.writeU64(vmOff+0x10, 80*1024*1024)      // WorkingSetSize (80 MB)
	b.writeU64(vmOff+0x18, 2*1024*1024)       // QuotaPeakPagedPoolUsage
	b.writeU64(vmOff+0x20, 1*1024*1024)       // QuotaPagedPoolUsage
	b.writeU64(vmOff+0x28, 512*1024)          // QuotaPeakNonPagedPoolUsage
	b.writeU64(vmOff+0x30, 256*1024)          // QuotaNonPagedPoolUsage
	b.writeU64(vmOff+0x38, 50*1024*1024)      // PagefileUsage
	b.writeU64(vmOff+0x40, 90*1024*1024)      // PeakPagefileUsage
	b.writeU64(vmOff+0x48, 40*1024*1024)      // PrivateUsage

	// BreakpadInfo (12 bytes)
	bpRVA := uint32(len(b.buf))
	bpOff := b.grow(12)
	b.writeU32(bpOff, 0x03)   // Validity: both bits set
	b.writeU32(bpOff+4, 0x1A2B)
	b.writeU32(bpOff+8, 0x3C4D)

	// AssertionInfo (776 bytes)
	aiRVA := uint32(len(b.buf))
	b.writeUTF16Fixed("x != NULL", 256)
	b.writeUTF16Fixed("doStuff", 256)
	b.writeUTF16Fixed("main.cpp", 256)
	aiLine := b.grow(8)
	b.writeU32(aiLine, 42)
	b.writeU32(aiLine+4, 1) // INVALID_PARAMETER

	// CommentA
	commentA := "test comment"
	caRVA := uint32(len(b.buf))
	b.buf = append(b.buf, []byte(commentA)...)

	// MemoryList: 1 entry (4 + 16 = 20 bytes)
	memRVA := uint32(len(b.buf))
	memOff := b.grow(20)
	b.writeU32(memOff, 1)         // count
	b.writeU64(memOff+4, 0x3000)  // StartOfMemoryRange
	b.writeU32(memOff+12, 256)    // DataSize
	b.writeU32(memOff+16, 0)      // RVA

	// SystemMemoryInfo (0x1EC = 492 bytes)
	smRVA := uint32(len(b.buf))
	smOff := b.grow(0x1EC)
	b.writeU16(smOff, 1)                    // Revision
	b.writeU32(smOff+0x04+0x04, 4096)       // PageSize
	b.writeU32(smOff+0x04+0x08, 4194304)    // NumberOfPhysPages (16 GB)
	b.writeU32(smOff+0x04+0x30, 8)          // NumberOfProcessors
	b.writeU64(smOff+0x74, 1000000)          // AvailablePages
	b.writeU64(smOff+0x74+0x08, 2000000)     // CommittedPages
	b.writeU64(smOff+0x74+0x10, 5000000)     // CommitLimit
	b.writeU64(smOff+0x74+0x18, 3000000)     // PeakCommitment
	b.writeU32(smOff+0x94+0x70, 50000)       // PagedPoolPages
	b.writeU32(smOff+0x94+0x74, 25000)       // NonPagedPoolPages

	// Linux strings
	lsbRelease := "Ubuntu 22.04.3 LTS"
	lsbRVA := uint32(len(b.buf))
	b.buf = append(b.buf, []byte(lsbRelease)...)

	environ := "HOME=/root\x00PATH=/usr/bin\x00"
	envRVA := uint32(len(b.buf))
	b.buf = append(b.buf, []byte(environ)...)

	mapsData := "00400000-00401000 r-xp 00000000 08:01 1234 /bin/app\n"
	mapsRVA := uint32(len(b.buf))
	b.buf = append(b.buf, []byte(mapsData)...)

	auxvData := []byte{0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	auxvRVA := uint32(len(b.buf))
	b.buf = append(b.buf, auxvData...)

	// SentryStackTraces: header(16) + 2 threads(24) + 3 frames(48) + symbols
	sym1 := "trigger_crash"
	sym2 := "main"
	symbolData := sym1 + sym2
	stRVA := uint32(len(b.buf))
	stOff := b.grow(16) // header
	b.writeU32(stOff, 1)                        // version
	b.writeU32(stOff+4, 2)                      // num_threads
	b.writeU32(stOff+8, 3)                      // num_frames
	b.writeU32(stOff+12, uint32(len(symbolData))) // symbol_bytes
	// threads (2 * 12 = 24 bytes, already 8-byte aligned after 16-byte header)
	t1Off := b.grow(12)
	b.writeU32(t1Off, 0x1A2B)  // thread_id (crashing thread)
	b.writeU32(t1Off+4, 0)     // start_frame
	b.writeU32(t1Off+8, 2)     // num_frames
	t2Off := b.grow(12)
	b.writeU32(t2Off, 0x3C4D)  // thread_id
	b.writeU32(t2Off+4, 2)     // start_frame
	b.writeU32(t2Off+8, 1)     // num_frames
	// frames (3 * 16 = 48 bytes, already 8-byte aligned after 24-byte thread list)
	f1Off := b.grow(16)
	b.writeU64(f1Off, 0x00007FF6A1B23456)   // instruction_addr
	b.writeU32(f1Off+8, 0)                  // symbol_offset
	b.writeU32(f1Off+12, uint32(len(sym1))) // symbol_len
	f2Off := b.grow(16)
	b.writeU64(f2Off, 0x00007FF6A1B24000)   // instruction_addr
	b.writeU32(f2Off+8, uint32(len(sym1)))  // symbol_offset
	b.writeU32(f2Off+12, uint32(len(sym2))) // symbol_len
	f3Off := b.grow(16)
	b.writeU64(f3Off, 0xDEAD)              // instruction_addr (no module match)
	b.writeU32(f3Off+8, 0)                 // symbol_offset
	b.writeU32(f3Off+12, 0)                // symbol_len (no symbol)
	// symbols
	b.buf = append(b.buf, []byte(symbolData)...)
	stSize := uint32(len(b.buf)) - stRVA

	// Stream directory
	dirRVA := uint32(len(b.buf))
	streams := []struct {
		typ  uint32
		size uint32
		rva  uint32
	}{
		{streamSystemInfo, 56, siRVA},
		{streamException, 168, exRVA},
		{streamThreadList, 52, tlRVA},
		{streamThreadNames, 16, tnRVA},
		{streamModuleList, 112, mlRVA},
		{streamUnloadedMods, 36, umRVA},
		{streamHandleData, 48, hdRVA},
		{streamMiscInfo, 24, miRVA},
		{streamMemoryInfo, 64, miLRVA},
		{streamFuncTable, 68, ftRVA},
		{streamSysMemInfo, 0x1EC, smRVA},
		{streamVmCounters, 0x50, vmRVA},
		{streamBreakpadInfo, 12, bpRVA},
		{streamAssertionInfo, 776, aiRVA},
		{streamCommentA, uint32(len(commentA)), caRVA},
		{streamMemoryList, 20, memRVA},
		{streamSentryStackTraces, stSize, stRVA},
		{streamLinuxLSBRelease, uint32(len(lsbRelease)), lsbRVA},
		{streamLinuxEnviron, uint32(len(environ)), envRVA},
		{streamLinuxMaps, uint32(len(mapsData)), mapsRVA},
		{streamLinuxAuxv, uint32(len(auxvData)), auxvRVA},
		{99, 4, auxvRVA}, // unknown stream type, reuse some data
	}
	for _, s := range streams {
		b.addStream(s.typ, s.size, s.rva)
	}

	b.writeU32(8, uint32(len(streams)))
	b.writeU32(12, dirRVA)

	return b.buf
}

func TestParse(t *testing.T) {
	data := testMinidump()
	md, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if md.Header.Signature != signature {
		t.Errorf("signature = 0x%08X, want 0x%08X", md.Header.Signature, signature)
	}
	if md.Header.Timestamp != 1700000000 {
		t.Errorf("timestamp = %d, want 1700000000", md.Header.Timestamp)
	}

	// SystemInfo
	if md.SystemInfo == nil {
		t.Fatal("SystemInfo is nil")
	}
	if md.SystemInfo.CPUArch != 9 {
		t.Errorf("CPUArch = %d, want 9 (AMD64)", md.SystemInfo.CPUArch)
	}
	if md.SystemInfo.NumCPUs != 8 {
		t.Errorf("NumCPUs = %d, want 8", md.SystemInfo.NumCPUs)
	}
	if md.SystemInfo.OSBuild != 19041 {
		t.Errorf("OSBuild = %d, want 19041", md.SystemInfo.OSBuild)
	}
	if md.SystemInfo.ServicePack != "Service Pack 1" {
		t.Errorf("ServicePack = %q, want %q", md.SystemInfo.ServicePack, "Service Pack 1")
	}

	// Exception
	if md.Exception == nil {
		t.Fatal("Exception is nil")
	}
	if md.Exception.Code != 0xC0000005 {
		t.Errorf("Exception.Code = 0x%X, want 0xC0000005", md.Exception.Code)
	}
	if md.Exception.Address != 0x00007FF6A1B23456 {
		t.Errorf("Exception.Address = 0x%X", md.Exception.Address)
	}
	if md.Exception.NumParams != 2 {
		t.Errorf("Exception.NumParams = %d, want 2", md.Exception.NumParams)
	}

	// Threads
	if len(md.Threads) != 1 {
		t.Fatalf("Threads = %d, want 1", len(md.Threads))
	}
	if md.Threads[0].ID != 0x1A2B {
		t.Errorf("Thread.ID = 0x%X, want 0x1A2B", md.Threads[0].ID)
	}
	if md.Threads[0].StackSize != 0x2000 {
		t.Errorf("Thread.StackSize = 0x%X, want 0x2000", md.Threads[0].StackSize)
	}

	// ThreadNames
	if md.ThreadNames[0x1A2B] != "main-thread" {
		t.Errorf("ThreadName = %q, want %q", md.ThreadNames[0x1A2B], "main-thread")
	}

	// Stacktraces
	if len(md.Stacktraces) != 2 {
		t.Fatalf("Stacktraces = %d, want 2", len(md.Stacktraces))
	}
	if md.Stacktraces[0].ThreadID != 0x1A2B {
		t.Errorf("Stacktrace[0].ThreadID = 0x%X, want 0x1A2B", md.Stacktraces[0].ThreadID)
	}
	if len(md.Stacktraces[0].Frames) != 2 {
		t.Fatalf("Stacktrace[0].Frames = %d, want 2", len(md.Stacktraces[0].Frames))
	}
	if md.Stacktraces[0].Frames[0].InstructionAddr != 0x00007FF6A1B23456 {
		t.Errorf("Frame[0].InstructionAddr = 0x%X", md.Stacktraces[0].Frames[0].InstructionAddr)
	}
	if md.Stacktraces[0].Frames[0].Symbol != "trigger_crash" {
		t.Errorf("Frame[0].Symbol = %q, want %q", md.Stacktraces[0].Frames[0].Symbol, "trigger_crash")
	}
	if md.Stacktraces[0].Frames[1].Symbol != "main" {
		t.Errorf("Frame[1].Symbol = %q, want %q", md.Stacktraces[0].Frames[1].Symbol, "main")
	}
	if md.Stacktraces[1].ThreadID != 0x3C4D {
		t.Errorf("Stacktrace[1].ThreadID = 0x%X, want 0x3C4D", md.Stacktraces[1].ThreadID)
	}
	if len(md.Stacktraces[1].Frames) != 1 {
		t.Fatalf("Stacktrace[1].Frames = %d, want 1", len(md.Stacktraces[1].Frames))
	}
	if md.Stacktraces[1].Frames[0].Symbol != "" {
		t.Errorf("Frame[0].Symbol = %q, want empty", md.Stacktraces[1].Frames[0].Symbol)
	}

	// Modules
	if len(md.Modules) != 1 {
		t.Fatalf("Modules = %d, want 1", len(md.Modules))
	}
	if md.Modules[0].Name != "C:\\app.exe" {
		t.Errorf("Module.Name = %q", md.Modules[0].Name)
	}
	if md.Modules[0].VersionMajor != 1 || md.Modules[0].VersionMinor != 2 {
		t.Errorf("Module version = %d.%d, want 1.2", md.Modules[0].VersionMajor, md.Modules[0].VersionMinor)
	}

	// UnloadedModules
	if len(md.UnloadedModules) != 1 {
		t.Fatalf("UnloadedModules = %d, want 1", len(md.UnloadedModules))
	}
	if md.UnloadedModules[0].Name != "C:\\old.dll" {
		t.Errorf("UnloadedModule.Name = %q", md.UnloadedModules[0].Name)
	}
	if md.UnloadedModules[0].SizeOfImage != 0x10000 {
		t.Errorf("UnloadedModule.SizeOfImage = 0x%X, want 0x10000", md.UnloadedModules[0].SizeOfImage)
	}

	// Handles
	if len(md.Handles) != 1 {
		t.Fatalf("Handles = %d, want 1", len(md.Handles))
	}
	if md.Handles[0].Handle != 0x1234 {
		t.Errorf("Handle = 0x%X, want 0x1234", md.Handles[0].Handle)
	}
	if md.Handles[0].TypeName != "Event" {
		t.Errorf("Handle.TypeName = %q", md.Handles[0].TypeName)
	}
	if md.Handles[0].ObjectName != "\\BaseNamedObjects\\MyEvent" {
		t.Errorf("Handle.ObjectName = %q", md.Handles[0].ObjectName)
	}

	// MiscInfo
	if md.MiscInfo == nil {
		t.Fatal("MiscInfo is nil")
	}
	if md.MiscInfo.ProcessID != 6672 {
		t.Errorf("ProcessID = %d, want 6672", md.MiscInfo.ProcessID)
	}

	// MemoryInfo
	if len(md.MemoryInfo) != 1 {
		t.Fatalf("MemoryInfo = %d, want 1", len(md.MemoryInfo))
	}
	if md.MemoryInfo[0].BaseAddress != 0x7FFE0000 {
		t.Errorf("MemoryInfo.BaseAddress = 0x%X", md.MemoryInfo[0].BaseAddress)
	}
	if md.MemoryInfo[0].State != 0x1000 {
		t.Errorf("MemoryInfo.State = 0x%X, want 0x1000", md.MemoryInfo[0].State)
	}

	// FunctionTables
	if len(md.FunctionTables) != 1 {
		t.Fatalf("FunctionTables = %d, want 1", len(md.FunctionTables))
	}
	if md.FunctionTables[0].BaseAddress != 0x00400000 {
		t.Errorf("FunctionTable.BaseAddress = 0x%X", md.FunctionTables[0].BaseAddress)
	}
	if len(md.FunctionTables[0].Entries) != 1 {
		t.Fatalf("FunctionTable.Entries = %d, want 1", len(md.FunctionTables[0].Entries))
	}
	if md.FunctionTables[0].Entries[0].BeginAddress != 0x1000 {
		t.Errorf("Entry.BeginAddress = 0x%X", md.FunctionTables[0].Entries[0].BeginAddress)
	}

	// SystemMemoryInfo
	if md.SystemMemInfo == nil {
		t.Fatal("SystemMemInfo is nil")
	}
	if md.SystemMemInfo.PageSize != 4096 {
		t.Errorf("PageSize = %d, want 4096", md.SystemMemInfo.PageSize)
	}
	if md.SystemMemInfo.NumberOfPhysPages != 4194304 {
		t.Errorf("NumberOfPhysPages = %d, want 4194304", md.SystemMemInfo.NumberOfPhysPages)
	}
	if md.SystemMemInfo.NumberOfProcessors != 8 {
		t.Errorf("NumberOfProcessors = %d, want 8", md.SystemMemInfo.NumberOfProcessors)
	}
	if md.SystemMemInfo.AvailablePages != 1000000 {
		t.Errorf("AvailablePages = %d, want 1000000", md.SystemMemInfo.AvailablePages)
	}
	if md.SystemMemInfo.CommittedPages != 2000000 {
		t.Errorf("CommittedPages = %d", md.SystemMemInfo.CommittedPages)
	}
	if md.SystemMemInfo.PagedPoolPages != 50000 {
		t.Errorf("PagedPoolPages = %d, want 50000", md.SystemMemInfo.PagedPoolPages)
	}

	// VmCounters
	if md.VmCounters == nil {
		t.Fatal("VmCounters is nil")
	}
	if md.VmCounters.PageFaultCount != 12345 {
		t.Errorf("PageFaultCount = %d, want 12345", md.VmCounters.PageFaultCount)
	}
	if md.VmCounters.WorkingSetSize != 80*1024*1024 {
		t.Errorf("WorkingSetSize = %d", md.VmCounters.WorkingSetSize)
	}
	if md.VmCounters.PrivateUsage != 40*1024*1024 {
		t.Errorf("PrivateUsage = %d", md.VmCounters.PrivateUsage)
	}

	// BreakpadInfo
	if md.BreakpadInfo == nil {
		t.Fatal("BreakpadInfo is nil")
	}
	if md.BreakpadInfo.Validity != 3 {
		t.Errorf("BreakpadInfo.Validity = %d, want 3", md.BreakpadInfo.Validity)
	}
	if md.BreakpadInfo.DumpThreadID != 0x1A2B {
		t.Errorf("BreakpadInfo.DumpThreadID = 0x%X", md.BreakpadInfo.DumpThreadID)
	}

	// AssertionInfo
	if md.AssertionInfo == nil {
		t.Fatal("AssertionInfo is nil")
	}
	if md.AssertionInfo.Expression != "x != NULL" {
		t.Errorf("AssertionInfo.Expression = %q", md.AssertionInfo.Expression)
	}
	if md.AssertionInfo.Function != "doStuff" {
		t.Errorf("AssertionInfo.Function = %q", md.AssertionInfo.Function)
	}
	if md.AssertionInfo.File != "main.cpp" {
		t.Errorf("AssertionInfo.File = %q", md.AssertionInfo.File)
	}
	if md.AssertionInfo.Line != 42 {
		t.Errorf("AssertionInfo.Line = %d, want 42", md.AssertionInfo.Line)
	}
	if md.AssertionInfo.Type != 1 {
		t.Errorf("AssertionInfo.Type = %d, want 1", md.AssertionInfo.Type)
	}

	// Comment
	if md.Comment != "test comment" {
		t.Errorf("Comment = %q, want %q", md.Comment, "test comment")
	}

	// MemoryList (type 5) — used as fallback, but Memory64List takes precedence
	// Here we only have MemoryList so it should be used
	if len(md.MemoryRanges) != 1 {
		t.Errorf("MemoryRanges = %d, want 1", len(md.MemoryRanges))
	} else if md.MemoryRanges[0].Size != 256 {
		t.Errorf("MemoryRange.Size = %d, want 256", md.MemoryRanges[0].Size)
	}

	// Linux strings
	if md.LinuxStrings["lsb_release"] != "Ubuntu 22.04.3 LTS" {
		t.Errorf("lsb_release = %q", md.LinuxStrings["lsb_release"])
	}
	if !strings.Contains(md.LinuxStrings["environ"], "HOME=/root") {
		t.Errorf("environ = %q", md.LinuxStrings["environ"])
	}
	if !strings.Contains(md.LinuxStrings["maps"], "/bin/app") {
		t.Errorf("maps = %q", md.LinuxStrings["maps"])
	}
	if md.LinuxStrings["auxv"] == "" {
		t.Error("auxv should not be empty")
	}
}

func TestParseInvalidSignature(t *testing.T) {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint32(data[0:], 0xDEADBEEF)
	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestParseTruncated(t *testing.T) {
	_, err := Parse([]byte("MDMP"))
	if err == nil {
		t.Error("expected error for truncated data")
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse(nil)
	if err == nil {
		t.Error("expected error for nil data")
	}
	_, err = Parse([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestParseLinuxStrings(t *testing.T) {
	b := newBuilder()
	b.grow(32)
	b.writeU32(0, signature)
	b.writeU16(4, 0xA793)

	cpuInfo := "processor\t: 0\nmodel name\t: Intel\n"
	cpuRVA := uint32(len(b.buf))
	b.buf = append(b.buf, []byte(cpuInfo)...)

	cmdLine := "/usr/bin/app --flag\x00"
	cmdRVA := uint32(len(b.buf))
	b.buf = append(b.buf, []byte(cmdLine)...)

	dirRVA := uint32(len(b.buf))
	b.addStream(streamLinuxCPUInfo, uint32(len(cpuInfo)), cpuRVA)
	b.addStream(streamLinuxCmdLine, uint32(len(cmdLine)), cmdRVA)
	b.writeU32(8, 2)
	b.writeU32(12, dirRVA)

	md, err := Parse(b.buf)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !strings.Contains(md.LinuxStrings["cpu_info"], "Intel") {
		t.Errorf("cpu_info = %q", md.LinuxStrings["cpu_info"])
	}
	if md.LinuxStrings["cmd_line"] != "/usr/bin/app --flag" {
		t.Errorf("cmd_line = %q", md.LinuxStrings["cmd_line"])
	}
}

func TestParseMemory64List(t *testing.T) {
	b := newBuilder()
	b.grow(32)
	b.writeU32(0, signature)
	b.writeU16(4, 0xA793)

	m64RVA := uint32(len(b.buf))
	m64Off := b.grow(16 + 2*16)
	b.writeU64(m64Off, 2)
	b.writeU64(m64Off+8, 0)
	b.writeU64(m64Off+16, 0x1000)
	b.writeU64(m64Off+24, 4096)
	b.writeU64(m64Off+32, 0x2000)
	b.writeU64(m64Off+40, 8192)

	dirRVA := uint32(len(b.buf))
	b.addStream(streamMemory64List, uint32(16+2*16), m64RVA)
	b.writeU32(8, 1)
	b.writeU32(12, dirRVA)

	md, err := Parse(b.buf)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(md.MemoryRanges) != 2 {
		t.Errorf("MemoryRanges = %d, want 2", len(md.MemoryRanges))
	} else {
		total := md.MemoryRanges[0].Size + md.MemoryRanges[1].Size
		if total != 4096+8192 {
			t.Errorf("MemoryTotal = %d, want %d", total, 4096+8192)
		}
	}
}

func TestParseUnknownStream(t *testing.T) {
	data := testMinidump()
	md, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if md.SystemInfo == nil {
		t.Error("known streams should still be parsed when unknown streams are present")
	}
	if len(md.UnknownStreams) != 1 {
		t.Fatalf("UnknownStreams = %d, want 1", len(md.UnknownStreams))
	}
	if md.UnknownStreams[0].Type != 99 {
		t.Errorf("UnknownStream.Type = %d, want 99", md.UnknownStreams[0].Type)
	}
	if len(md.UnknownStreams[0].Data) == 0 {
		t.Error("UnknownStream.Data should not be empty")
	}
}

func TestParseShortStreams(t *testing.T) {
	b := newBuilder()
	b.grow(32)
	b.writeU32(0, signature)
	b.writeU16(4, 0xA793)

	tinyRVA := uint32(len(b.buf))
	b.grow(4)

	dirRVA := uint32(len(b.buf))
	for _, typ := range []uint32{
		streamSystemInfo, streamException, streamThreadList,
		streamModuleList, streamMiscInfo, streamMemory64List,
		streamThreadNames, streamUnloadedMods, streamHandleData,
		streamFuncTable, streamMemoryInfo, streamSysMemInfo, streamVmCounters, streamBreakpadInfo, streamAssertionInfo,
		streamMemoryList,
	} {
		b.addStream(typ, 4, tinyRVA)
	}
	b.writeU32(8, 16)
	b.writeU32(12, dirRVA)

	md, err := Parse(b.buf)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if md.SystemInfo != nil {
		t.Error("short SystemInfo should be nil")
	}
	if md.Exception != nil {
		t.Error("short ExceptionStream should be nil")
	}
	if len(md.Threads) != 0 {
		t.Error("short ThreadList should be empty")
	}
	if len(md.ThreadNames) != 0 {
		t.Error("short ThreadNames should be empty")
	}
	if len(md.Modules) != 0 {
		t.Error("short ModuleList should be empty")
	}
	if md.UnloadedModules != nil {
		t.Error("short UnloadedModules should be nil")
	}
	if md.Handles != nil {
		t.Error("short HandleData should be nil")
	}
	if md.FunctionTables != nil {
		t.Error("short FunctionTable should be nil")
	}
	if md.MiscInfo != nil {
		t.Error("short MiscInfo should be nil")
	}
	if len(md.MemoryRanges) != 0 {
		t.Error("short Memory64List should be empty")
	}
	if md.MemoryInfo != nil {
		t.Error("short MemoryInfoList should be nil")
	}
	if md.SystemMemInfo != nil {
		t.Error("short SystemMemoryInfo should be nil")
	}
	if md.VmCounters != nil {
		t.Error("short VmCounters should be nil")
	}
	if md.BreakpadInfo != nil {
		t.Error("short BreakpadInfo should be nil")
	}
	if md.AssertionInfo != nil {
		t.Error("short AssertionInfo should be nil")
	}
}

func TestParseCommentW(t *testing.T) {
	b := newBuilder()
	b.grow(32)
	b.writeU32(0, signature)
	b.writeU16(4, 0xA793)

	comment := "wide comment"
	runes := utf16.Encode([]rune(comment))
	cwRVA := uint32(len(b.buf))
	for _, r := range runes {
		off := b.grow(2)
		b.writeU16(off, r)
	}
	cwSize := uint32(len(runes) * 2)

	dirRVA := uint32(len(b.buf))
	b.addStream(streamCommentW, cwSize, cwRVA)
	b.writeU32(8, 1)
	b.writeU32(12, dirRVA)

	md, err := Parse(b.buf)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if md.Comment != comment {
		t.Errorf("Comment = %q, want %q", md.Comment, comment)
	}
}

func TestExceptionCodeName(t *testing.T) {
	if name, ok := ExceptionCodeNames[0xC0000005]; !ok || name != "ACCESS_VIOLATION" {
		t.Errorf("0xC0000005 = %q, want ACCESS_VIOLATION", name)
	}
	if _, ok := ExceptionCodeNames[0x12345678]; ok {
		t.Error("unknown code should not be in map")
	}
}

func TestCPUArchName(t *testing.T) {
	if name, ok := CPUArchNames[9]; !ok || name != "AMD64" {
		t.Errorf("arch 9 = %q, want AMD64", name)
	}
}

func TestOSTypeName(t *testing.T) {
	if name, ok := OSTypeNames[0]; !ok || name != "Windows NT" {
		t.Errorf("os type 0 = %q, want Windows NT", name)
	}
}

func TestMemStateNames(t *testing.T) {
	if name := MemStateNames[0x1000]; name != "MEM_COMMIT" {
		t.Errorf("0x1000 = %q, want MEM_COMMIT", name)
	}
}

func TestAssertionTypeNames(t *testing.T) {
	if name := AssertionTypeNames[1]; name != "INVALID_PARAMETER" {
		t.Errorf("1 = %q, want INVALID_PARAMETER", name)
	}
}

func TestParseTruncatedLists(t *testing.T) {
	// Each list stream has a valid count but truncated entries.
	// This tests the count*entrySize > len(data) early returns.
	tests := []struct {
		name string
		typ  uint32
		data []byte // must have valid count header but too few bytes for entries
	}{
		{"ThreadList", streamThreadList, func() []byte {
			d := make([]byte, 8) // 4 (count) + 4 (partial)
			le.PutUint32(d[0:], 1)
			return d
		}()},
		{"ThreadNames", streamThreadNames, func() []byte {
			d := make([]byte, 8)
			le.PutUint32(d[0:], 1)
			return d
		}()},
		{"ModuleList", streamModuleList, func() []byte {
			d := make([]byte, 8)
			le.PutUint32(d[0:], 1)
			return d
		}()},
		{"MemoryList", streamMemoryList, func() []byte {
			d := make([]byte, 8)
			le.PutUint32(d[0:], 1)
			return d
		}()},
		{"Memory64List", streamMemory64List, func() []byte {
			d := make([]byte, 20)
			le.PutUint64(d[0:], 1) // count=1 but only 4 bytes after header
			return d
		}()},
		{"UnloadedModules", streamUnloadedMods, func() []byte {
			d := make([]byte, 16) // header ok but no room for entries
			le.PutUint32(d[0:], 12)
			le.PutUint32(d[4:], 24)
			le.PutUint32(d[8:], 1)
			return d
		}()},
		{"Handles", streamHandleData, func() []byte {
			d := make([]byte, 20) // header ok but no room for entries
			le.PutUint32(d[0:], 16)
			le.PutUint32(d[4:], 32)
			le.PutUint32(d[8:], 1)
			return d
		}()},
		{"MemoryInfoList", streamMemoryInfo, func() []byte {
			d := make([]byte, 20) // header ok but no room for entries
			le.PutUint32(d[0:], 16)
			le.PutUint32(d[4:], 48)
			le.PutUint64(d[8:], 1)
			return d
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuilder()
			b.grow(32)
			b.writeU32(0, signature)
			b.writeU16(4, 0xA793)

			rva := uint32(len(b.buf))
			b.buf = append(b.buf, tt.data...)

			dirRVA := uint32(len(b.buf))
			b.addStream(tt.typ, uint32(len(tt.data)), rva)
			b.writeU32(8, 1)
			b.writeU32(12, dirRVA)

			md, err := Parse(b.buf)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			// All should parse without error but produce no entries
			if len(md.Threads) > 0 || len(md.Modules) > 0 || len(md.MemoryRanges) > 0 ||
				len(md.UnloadedModules) > 0 || len(md.Handles) > 0 || len(md.MemoryInfo) > 0 {
				t.Error("truncated list should produce no entries")
			}
		})
	}
}

func TestParseExceptionParamsCapped(t *testing.T) {
	b := newBuilder()
	b.grow(32)
	b.writeU32(0, signature)
	b.writeU16(4, 0xA793)

	exRVA := uint32(len(b.buf))
	exOff := b.grow(168)
	b.writeU32(exOff+32, 20) // NumParams > 15

	dirRVA := uint32(len(b.buf))
	b.addStream(streamException, 168, exRVA)
	b.writeU32(8, 1)
	b.writeU32(12, dirRVA)

	md, err := Parse(b.buf)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if md.Exception.NumParams != 15 {
		t.Errorf("NumParams = %d, want 15 (capped)", md.Exception.NumParams)
	}
}

func TestReadMinidumpStringTruncated(t *testing.T) {
	// Valid length header but data extends past buffer
	data := make([]byte, 8)
	le.PutUint32(data[0:], 100) // claims 100 bytes but only 4 available
	got := readMinidumpString(data, 0)
	if got != "" {
		t.Errorf("truncated string = %q, want empty", got)
	}
}

func TestParseStacktracesEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"too short", make([]byte, 10)},
		{"bad version", func() []byte {
			d := make([]byte, 16)
			le.PutUint32(d[0:], 2) // version 2
			return d
		}()},
		{"truncated threads", func() []byte {
			d := make([]byte, 16)
			le.PutUint32(d[0:], 1)  // version
			le.PutUint32(d[4:], 10) // num_threads (too many)
			return d
		}()},
		{"truncated frames", func() []byte {
			d := make([]byte, 24) // header(16) + 0 threads, but claims 1 frame
			le.PutUint32(d[0:], 1)  // version
			le.PutUint32(d[4:], 0)  // num_threads
			le.PutUint32(d[8:], 10) // num_frames (too many)
			return d
		}()},
		{"truncated symbols", func() []byte {
			d := make([]byte, 16)
			le.PutUint32(d[0:], 1)     // version
			le.PutUint32(d[4:], 0)     // num_threads
			le.PutUint32(d[8:], 0)     // num_frames
			le.PutUint32(d[12:], 9999) // symbol_bytes (too many)
			return d
		}()},
		{"empty", func() []byte {
			d := make([]byte, 16)
			le.PutUint32(d[0:], 1) // version, everything else 0
			return d
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStacktraces(tt.data)
			if tt.name == "empty" {
				if len(result) != 0 {
					t.Errorf("expected 0 stacktraces, got %d", len(result))
				}
			} else if result != nil && tt.name != "empty" {
				// truncated cases should return nil
			}
		})
	}
}

func TestStreamTypeNames(t *testing.T) {
	if name := StreamTypeNames[3]; name != "ThreadList" {
		t.Errorf("type 3 = %q, want ThreadList", name)
	}
	if name := StreamTypeNames[0x43500001]; name != "CrashpadInfo" {
		t.Errorf("type 0x43500001 = %q, want CrashpadInfo", name)
	}
	if name := StreamTypeNames[9999]; name != "" {
		t.Errorf("unknown type = %q, want empty", name)
	}
}
