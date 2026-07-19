package minidump

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf16"
)

const signature = 0x504D444D // "MDMP" in little-endian

const (
	streamUnused              = 0
	streamReserved0           = 1
	streamReserved1           = 2
	streamThreadList          = 3
	streamModuleList          = 4
	streamMemoryList          = 5
	streamException           = 6
	streamSystemInfo          = 7
	streamThreadExList        = 8
	streamMemory64List        = 9
	streamCommentA            = 10
	streamCommentW            = 11
	streamHandleData          = 12
	streamFuncTable           = 13
	streamUnloadedMods        = 14
	streamMiscInfo            = 15
	streamMemoryInfo          = 16
	streamThreadInfo          = 17
	streamHandleOperationList = 18
	streamToken               = 19
	streamJavaScriptData      = 20
	streamSysMemInfo          = 21
	streamVmCounters          = 22
	streamIptTrace            = 23
	streamThreadNames         = 24
	streamCompressedMemory    = 25
	streamCompressedMemorySQL = 26

	ceStreamNull                = 0x8000
	ceStreamSystemInfo          = 0x8001
	ceStreamException           = 0x8002
	ceStreamModuleList          = 0x8003
	ceStreamProcessList         = 0x8004
	ceStreamThreadList          = 0x8005
	ceStreamThreadContextList   = 0x8006
	ceStreamThreadCallStackList = 0x8007
	ceStreamMemoryVirtualList   = 0x8008
	ceStreamMemoryPhysicalList  = 0x8009
	ceStreamBucketParameters    = 0x800A
	ceStreamProcessModuleMap    = 0x800B
	ceStreamDiagnosisList       = 0x800C
	streamLastReserved          = 0xFFFF

	streamCrashpadInfo      = 0x43500001
	streamSentryStackTraces = 0x53790001

	streamBreakpadInfo    = 0x47670001
	streamAssertionInfo   = 0x47670002
	streamLinuxCPUInfo    = 0x47670003
	streamLinuxProcStatus = 0x47670004
	streamLinuxLSBRelease = 0x47670005
	streamLinuxCmdLine    = 0x47670006
	streamLinuxEnviron    = 0x47670007
	streamLinuxAuxv       = 0x47670008
	streamLinuxMaps       = 0x47670009
	streamLinuxDsoDebug   = 0x4767000A
)

type Minidump struct {
	Header          Header
	Streams         []Stream
	SystemInfo      *SystemInfo
	Exception       *Exception
	Threads         []Thread
	ThreadNames     map[uint32]string
	Stacktraces     []Stacktrace
	Modules         []Module
	UnloadedModules []UnloadedModule
	Handles         []Handle
	FunctionTables  []FunctionTable
	MiscInfo        *MiscInfo
	MemoryRanges    []MemoryRange
	MemoryInfo      []MemoryInfo
	SystemMemInfo   *SystemMemoryInfo
	VmCounters      *ProcessVmCounters
	BreakpadInfo    *BreakpadInfo
	AssertionInfo   *AssertionInfo
	Comment         string
	LinuxStrings    map[string]string
	UnknownStreams  []UnknownStream
}

type Stream struct {
	Type uint32
	Size uint32
}

type UnknownStream struct {
	Type uint32
	Data []byte
}

type Header struct {
	Signature    uint32
	Version      uint16
	ImplVersion  uint16
	StreamCount  uint32
	StreamDirRVA uint32
	Checksum     uint32
	Timestamp    uint32
	Flags        uint64
}

type SystemInfo struct {
	CPUArch     uint16
	CPULevel    uint16
	CPURevision uint16
	NumCPUs     uint8
	OSType      uint32
	OSVerMajor  uint32
	OSVerMinor  uint32
	OSBuild     uint32
	OSPlatform  uint32
	ServicePack string
	SuiteMask   uint16
}

type Exception struct {
	ThreadID  uint32
	Code      uint32
	Flags     uint32
	Address   uint64
	NumParams uint32
	Params    []uint64
}

type Thread struct {
	ID            uint32
	SuspendCount  uint32
	PriorityClass uint32
	Priority      uint32
	TEB           uint64
	StackStart    uint64
	StackSize     uint32
}

type Stacktrace struct {
	ThreadID uint32
	Frames   []StackFrame
}

type StackFrame struct {
	InstructionAddr uint64
	Symbol          string
}

type Module struct {
	BaseOfImage  uint64
	SizeOfImage  uint32
	Checksum     uint32
	Timestamp    uint32
	Name         string
	VersionMajor uint16
	VersionMinor uint16
	VersionBuild uint16
	VersionPatch uint16
}

type UnloadedModule struct {
	BaseOfImage uint64
	SizeOfImage uint32
	Checksum    uint32
	Timestamp   uint32
	Name        string
}

type Handle struct {
	Handle        uint64
	TypeName      string
	ObjectName    string
	Attributes    uint32
	GrantedAccess uint32
	HandleCount   uint32
	PointerCount  uint32
}

type FunctionTable struct {
	MinAddress  uint64
	MaxAddress  uint64
	BaseAddress uint64
	Entries     []FunctionEntry
}

type FunctionEntry struct {
	BeginAddress      uint32
	EndAddress        uint32
	UnwindInfoAddress uint32
}

type MemoryRange struct {
	Address uint64
	Size    uint64
}

type MemoryInfo struct {
	BaseAddress uint64
	RegionSize  uint64
	State       uint32
	Protect     uint32
	Type        uint32
}

type MiscInfo struct {
	ProcessID         uint32
	ProcessCreateTime uint32
	ProcessUserTime   uint32
	ProcessKernelTime uint32
}

type SystemMemoryInfo struct {
	PageSize           uint32
	NumberOfPhysPages  uint32
	NumberOfProcessors uint32
	AvailablePages     uint64
	CommittedPages     uint64
	CommitLimit        uint64
	PeakCommitment     uint64
	PagedPoolPages     uint32
	NonPagedPoolPages  uint32
}

type ProcessVmCounters struct {
	PageFaultCount             uint32
	PeakWorkingSetSize         uint64
	WorkingSetSize             uint64
	QuotaPeakPagedPoolUsage    uint64
	QuotaPagedPoolUsage        uint64
	QuotaPeakNonPagedPoolUsage uint64
	QuotaNonPagedPoolUsage     uint64
	PagefileUsage              uint64
	PeakPagefileUsage          uint64
	PrivateUsage               uint64
}

type BreakpadInfo struct {
	Validity           uint32
	DumpThreadID       uint32
	RequestingThreadID uint32
}

type AssertionInfo struct {
	Expression string
	Function   string
	File       string
	Line       uint32
	Type       uint32
}

type streamDirEntry struct {
	StreamType uint32
	DataSize   uint32
	DataRVA    uint32
}

var ExceptionCodeNames = map[uint32]string{
	0x40010006: "DBG_PRINTEXCEPTION_C",
	0x406D1388: "MS_VC_EXCEPTION",
	0x80000003: "BREAKPOINT",
	0x80000004: "SINGLE_STEP",
	0xC0000005: "ACCESS_VIOLATION",
	0xC000001D: "ILLEGAL_INSTRUCTION",
	0xC0000094: "INTEGER_DIVIDE_BY_ZERO",
	0xC0000096: "PRIVILEGED_INSTRUCTION",
	0xC00000FD: "STACK_OVERFLOW",
	0xC0000409: "STACK_BUFFER_OVERRUN",
}

var CPUArchNames = map[uint16]string{
	0:  "x86",
	5:  "ARM",
	6:  "IA64",
	9:  "AMD64",
	12: "ARM64",
}

var OSTypeNames = map[uint32]string{
	0: "Windows NT",
	1: "Windows CE",
	2: "macOS",
	3: "iOS",
	4: "Linux",
	5: "Solaris",
	6: "Android",
	7: "PS3",
	8: "NaCl",
}

var MemStateNames = map[uint32]string{
	0x1000:  "MEM_COMMIT",
	0x2000:  "MEM_RESERVE",
	0x10000: "MEM_FREE",
}

var MemTypeNames = map[uint32]string{
	0x20000:   "MEM_PRIVATE",
	0x40000:   "MEM_MAPPED",
	0x1000000: "MEM_IMAGE",
}

var MemProtectNames = map[uint32]string{
	0x01:  "NOACCESS",
	0x02:  "READONLY",
	0x04:  "READWRITE",
	0x08:  "WRITECOPY",
	0x10:  "EXECUTE",
	0x20:  "EXECUTE_READ",
	0x40:  "EXECUTE_READWRITE",
	0x80:  "EXECUTE_WRITECOPY",
	0x100: "GUARD",
	0x200: "NOCACHE",
	0x400: "WRITECOMBINE",
}

var StreamTypeNames = map[uint32]string{
	streamUnused:                "Unused",
	streamReserved0:             "Reserved0",
	streamReserved1:             "Reserved1",
	streamThreadList:            "ThreadList",
	streamModuleList:            "ModuleList",
	streamMemoryList:            "MemoryList",
	streamException:             "Exception",
	streamSystemInfo:            "SystemInfo",
	streamThreadExList:          "ThreadExList",
	streamMemory64List:          "Memory64List",
	streamCommentA:              "CommentA",
	streamCommentW:              "CommentW",
	streamHandleData:            "HandleData",
	streamFuncTable:             "FunctionTable",
	streamUnloadedMods:          "UnloadedModuleList",
	streamMiscInfo:              "MiscInfo",
	streamMemoryInfo:            "MemoryInfoList",
	streamThreadInfo:            "ThreadInfoList",
	streamHandleOperationList:   "HandleOperationList",
	streamToken:                 "Token",
	streamJavaScriptData:        "JavaScriptData",
	streamSysMemInfo:            "SystemMemoryInfo",
	streamVmCounters:            "ProcessVmCounters",
	streamIptTrace:              "IptTrace",
	streamThreadNames:           "ThreadNames",
	streamCompressedMemory:      "CompressedMemory",
	streamCompressedMemorySQL:   "CompressedMemorySQL",
	ceStreamNull:                "ceStreamNull",
	ceStreamSystemInfo:          "ceStreamSystemInfo",
	ceStreamException:           "ceStreamException",
	ceStreamModuleList:          "ceStreamModuleList",
	ceStreamProcessList:         "ceStreamProcessList",
	ceStreamThreadList:          "ceStreamThreadList",
	ceStreamThreadContextList:   "ceStreamThreadContextList",
	ceStreamThreadCallStackList: "ceStreamThreadCallStackList",
	ceStreamMemoryVirtualList:   "ceStreamMemoryVirtualList",
	ceStreamMemoryPhysicalList:  "ceStreamMemoryPhysicalList",
	ceStreamBucketParameters:    "ceStreamBucketParameters",
	ceStreamProcessModuleMap:    "ceStreamProcessModuleMap",
	ceStreamDiagnosisList:       "ceStreamDiagnosisList",
	streamLastReserved:          "LastReserved",
	streamBreakpadInfo:          "BreakpadInfo",
	streamAssertionInfo:         "AssertionInfo",
	streamLinuxCPUInfo:          "LinuxCpuInfo",
	streamLinuxProcStatus:       "LinuxProcStatus",
	streamLinuxLSBRelease:       "LinuxLsbRelease",
	streamLinuxCmdLine:          "LinuxCmdLine",
	streamLinuxEnviron:          "LinuxEnviron",
	streamLinuxAuxv:             "LinuxAuxv",
	streamLinuxMaps:             "LinuxMaps",
	streamLinuxDsoDebug:         "LinuxDsoDebug",
	streamCrashpadInfo:          "CrashpadInfo",
	streamSentryStackTraces:     "SentryStackTraces",
}

var AssertionTypeNames = map[uint32]string{
	0: "UNKNOWN",
	1: "INVALID_PARAMETER",
	2: "PURE_VIRTUAL_CALL",
}

func Parse(data []byte) (*Minidump, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("minidump: data too short (%d bytes)", len(data))
	}

	var hdr Header
	hdr.Signature = le.Uint32(data[0:])
	if hdr.Signature != signature {
		return nil, fmt.Errorf("minidump: invalid signature 0x%08X", hdr.Signature)
	}
	hdr.Version = le.Uint16(data[4:])
	hdr.ImplVersion = le.Uint16(data[6:])
	hdr.StreamCount = le.Uint32(data[8:])
	hdr.StreamDirRVA = le.Uint32(data[12:])
	hdr.Checksum = le.Uint32(data[16:])
	hdr.Timestamp = le.Uint32(data[20:])
	hdr.Flags = le.Uint64(data[24:])

	md := &Minidump{Header: hdr}

	dirEnd := int64(hdr.StreamDirRVA) + int64(hdr.StreamCount)*12
	if dirEnd > int64(len(data)) {
		return nil, fmt.Errorf("minidump: stream directory out of bounds")
	}

	for i := uint32(0); i < hdr.StreamCount; i++ {
		off := int(hdr.StreamDirRVA) + int(i)*12
		var entry streamDirEntry
		entry.StreamType = le.Uint32(data[off:])
		entry.DataSize = le.Uint32(data[off+4:])
		entry.DataRVA = le.Uint32(data[off+8:])
		md.Streams = append(md.Streams, Stream{
			Type: entry.StreamType,
			Size: entry.DataSize,
		})

		end := int64(entry.DataRVA) + int64(entry.DataSize)
		if end > int64(len(data)) {
			continue
		}
		stream := data[entry.DataRVA:end]
		if entry.DataSize == 0 {
			continue
		}

		switch entry.StreamType {
		case streamSystemInfo:
			md.SystemInfo = parseSystemInfo(stream, data)
		case streamException:
			md.Exception = parseException(stream)
		case streamThreadList:
			md.Threads = parseThreadList(stream)
		case streamThreadNames:
			md.ThreadNames = parseThreadNames(stream, data)
		case streamModuleList:
			md.Modules = parseModuleList(stream, data)
		case streamUnloadedMods:
			md.UnloadedModules = parseUnloadedModules(stream, data)
		case streamHandleData:
			md.Handles = parseHandles(stream, data)
		case streamFuncTable:
			md.FunctionTables = parseFunctionTables(stream)
		case streamMiscInfo:
			md.MiscInfo = parseMiscInfo(stream)
		case streamMemoryList:
			if md.MemoryRanges == nil {
				md.MemoryRanges = parseMemoryList(stream)
			}
		case streamMemory64List:
			md.MemoryRanges = parseMemory64List(stream)
		case streamMemoryInfo:
			md.MemoryInfo = parseMemoryInfoList(stream)
		case streamSysMemInfo:
			md.SystemMemInfo = parseSystemMemInfo(stream)
		case streamVmCounters:
			md.VmCounters = parseVmCounters(stream)
		case streamCommentA:
			md.Comment = strings.TrimRight(string(stream), "\x00")
		case streamCommentW:
			md.Comment = decodeUTF16(stream)
		case streamSentryStackTraces:
			md.Stacktraces = parseStacktraces(stream)
		case streamBreakpadInfo:
			md.BreakpadInfo = parseBreakpadInfo(stream)
		case streamAssertionInfo:
			md.AssertionInfo = parseAssertionInfo(stream)
		case streamLinuxCPUInfo:
			md.linuxString("cpu_info", stream)
		case streamLinuxProcStatus:
			md.linuxString("proc_status", stream)
		case streamLinuxLSBRelease:
			md.linuxString("lsb_release", stream)
		case streamLinuxCmdLine:
			md.linuxString("cmd_line", stream)
		case streamLinuxEnviron:
			md.linuxString("environ", strings.ReplaceAll(string(stream), "\x00", "\n"))
		case streamLinuxAuxv:
			md.linuxString("auxv", hex.Dump(stream))
		case streamLinuxMaps:
			md.linuxString("maps", stream)
		default:
			md.UnknownStreams = append(md.UnknownStreams, UnknownStream{
				Type: entry.StreamType,
				Data: stream,
			})
		}
	}

	return md, nil
}

func (md *Minidump) linuxString(key string, data any) {
	if md.LinuxStrings == nil {
		md.LinuxStrings = make(map[string]string)
	}
	switch v := data.(type) {
	case []byte:
		md.LinuxStrings[key] = strings.TrimRight(string(v), "\x00\n")
	case string:
		md.LinuxStrings[key] = strings.TrimRight(v, "\x00\n")
	}
}

func parseSystemInfo(data []byte, full []byte) *SystemInfo {
	if len(data) < 56 {
		return nil
	}
	si := &SystemInfo{
		CPUArch:     le.Uint16(data[0:]),
		CPULevel:    le.Uint16(data[2:]),
		CPURevision: le.Uint16(data[4:]),
		NumCPUs:     data[6],
		OSType:      le.Uint32(data[8:]),
		OSVerMajor:  le.Uint32(data[12:]),
		OSVerMinor:  le.Uint32(data[16:]),
		OSBuild:     le.Uint32(data[20:]),
		OSPlatform:  le.Uint32(data[24:]),
		SuiteMask:   le.Uint16(data[52:]),
	}
	spRVA := le.Uint32(data[28:])
	si.ServicePack = readMinidumpString(full, spRVA)
	return si
}

func parseException(data []byte) *Exception {
	if len(data) < 168 {
		return nil
	}
	ex := &Exception{
		ThreadID: le.Uint32(data[0:]),
		Code:     le.Uint32(data[8:]),
		Flags:    le.Uint32(data[12:]),
		Address:  le.Uint64(data[24:]),
	}
	ex.NumParams = le.Uint32(data[32:])
	if ex.NumParams > 15 {
		ex.NumParams = 15
	}
	ex.Params = make([]uint64, ex.NumParams)
	for i := uint32(0); i < ex.NumParams; i++ {
		ex.Params[i] = le.Uint64(data[40+i*8:])
	}
	return ex
}

func parseThreadList(data []byte) []Thread {
	if len(data) < 4 {
		return nil
	}
	count := le.Uint32(data[0:])
	if int64(4+count*48) > int64(len(data)) {
		return nil
	}
	threads := make([]Thread, count)
	for i := uint32(0); i < count; i++ {
		off := 4 + int(i)*48
		threads[i] = Thread{
			ID:            le.Uint32(data[off:]),
			SuspendCount:  le.Uint32(data[off+4:]),
			PriorityClass: le.Uint32(data[off+8:]),
			Priority:      le.Uint32(data[off+12:]),
			TEB:           le.Uint64(data[off+16:]),
			StackStart:    le.Uint64(data[off+24:]),
			StackSize:     le.Uint32(data[off+32:]),
		}
	}
	return threads
}

func parseThreadNames(data []byte, full []byte) map[uint32]string {
	if len(data) < 4 {
		return nil
	}
	count := le.Uint32(data[0:])
	if int64(4+count*12) > int64(len(data)) {
		return nil
	}
	names := make(map[uint32]string, count)
	for i := uint32(0); i < count; i++ {
		off := 4 + int(i)*12
		tid := le.Uint32(data[off:])
		nameRVA := le.Uint64(data[off+4:])
		if nameRVA <= 0xFFFFFFFF {
			name := readMinidumpString(full, uint32(nameRVA))
			if name != "" {
				names[tid] = name
			}
		}
	}
	return names
}

func parseModuleList(data []byte, full []byte) []Module {
	if len(data) < 4 {
		return nil
	}
	count := le.Uint32(data[0:])
	if int64(4+count*108) > int64(len(data)) {
		return nil
	}
	modules := make([]Module, count)
	for i := uint32(0); i < count; i++ {
		off := 4 + int(i)*108
		nameRVA := le.Uint32(data[off+24:])
		versionMS := le.Uint32(data[off+28:])
		versionLS := le.Uint32(data[off+32:])
		modules[i] = Module{
			BaseOfImage:  le.Uint64(data[off:]),
			SizeOfImage:  le.Uint32(data[off+8:]),
			Checksum:     le.Uint32(data[off+12:]),
			Timestamp:    le.Uint32(data[off+16:]),
			Name:         readMinidumpString(full, nameRVA),
			VersionMajor: uint16(versionMS >> 16),
			VersionMinor: uint16(versionMS),
			VersionBuild: uint16(versionLS >> 16),
			VersionPatch: uint16(versionLS),
		}
	}
	return modules
}

func parseUnloadedModules(data []byte, full []byte) []UnloadedModule {
	// Header: SizeOfHeader(4) + SizeOfEntry(4) + NumberOfEntries(4)
	if len(data) < 12 {
		return nil
	}
	headerSize := le.Uint32(data[0:])
	entrySize := le.Uint32(data[4:])
	count := le.Uint32(data[8:])
	if entrySize < 24 || int64(headerSize)+int64(count)*int64(entrySize) > int64(len(data)) {
		return nil
	}
	modules := make([]UnloadedModule, count)
	for i := uint32(0); i < count; i++ {
		off := int(headerSize) + int(i)*int(entrySize)
		modules[i] = UnloadedModule{
			BaseOfImage: le.Uint64(data[off:]),
			SizeOfImage: le.Uint32(data[off+8:]),
			Checksum:    le.Uint32(data[off+12:]),
			Timestamp:   le.Uint32(data[off+16:]),
			Name:        readMinidumpString(full, le.Uint32(data[off+20:])),
		}
	}
	return modules
}

func parseHandles(data []byte, full []byte) []Handle {
	// Header: SizeOfHeader(4) + SizeOfDescriptor(4) + NumberOfDescriptors(4) + Reserved(4)
	if len(data) < 16 {
		return nil
	}
	headerSize := le.Uint32(data[0:])
	descSize := le.Uint32(data[4:])
	count := le.Uint32(data[8:])
	if descSize < 32 || int64(headerSize)+int64(count)*int64(descSize) > int64(len(data)) {
		return nil
	}
	handles := make([]Handle, count)
	for i := uint32(0); i < count; i++ {
		off := int(headerSize) + int(i)*int(descSize)
		handles[i] = Handle{
			Handle:        le.Uint64(data[off:]),
			TypeName:      readMinidumpString(full, le.Uint32(data[off+8:])),
			ObjectName:    readMinidumpString(full, le.Uint32(data[off+12:])),
			Attributes:    le.Uint32(data[off+16:]),
			GrantedAccess: le.Uint32(data[off+20:]),
			HandleCount:   le.Uint32(data[off+24:]),
			PointerCount:  le.Uint32(data[off+28:]),
		}
	}
	return handles
}

func parseMiscInfo(data []byte) *MiscInfo {
	if len(data) < 24 {
		return nil
	}
	return &MiscInfo{
		ProcessID:         le.Uint32(data[8:]),
		ProcessCreateTime: le.Uint32(data[12:]),
		ProcessUserTime:   le.Uint32(data[16:]),
		ProcessKernelTime: le.Uint32(data[20:]),
	}
}

func parseMemoryList(data []byte) []MemoryRange {
	if len(data) < 4 {
		return nil
	}
	n := le.Uint32(data[0:])
	if int64(4+n*16) > int64(len(data)) {
		return nil
	}
	ranges := make([]MemoryRange, n)
	for i := uint32(0); i < n; i++ {
		off := 4 + int(i)*16
		ranges[i] = MemoryRange{
			Address: le.Uint64(data[off:]),
			Size:    uint64(le.Uint32(data[off+8:])),
		}
	}
	return ranges
}

func parseMemory64List(data []byte) []MemoryRange {
	if len(data) < 16 {
		return nil
	}
	count := int(le.Uint64(data[0:]))
	if int64(16+count*16) > int64(len(data)) {
		count = (len(data) - 16) / 16
	}
	ranges := make([]MemoryRange, count)
	for i := 0; i < count; i++ {
		off := 16 + i*16
		ranges[i] = MemoryRange{
			Address: le.Uint64(data[off:]),
			Size:    le.Uint64(data[off+8:]),
		}
	}
	return ranges
}

func parseMemoryInfoList(data []byte) []MemoryInfo {
	// Header: SizeOfHeader(4) + SizeOfEntry(4) + NumberOfEntries(8)
	if len(data) < 16 {
		return nil
	}
	headerSize := le.Uint32(data[0:])
	entrySize := le.Uint32(data[4:])
	count := le.Uint64(data[8:])
	if entrySize < 48 || int64(headerSize)+int64(count)*int64(entrySize) > int64(len(data)) {
		return nil
	}
	infos := make([]MemoryInfo, count)
	for i := uint64(0); i < count; i++ {
		off := int(headerSize) + int(i)*int(entrySize)
		infos[i] = MemoryInfo{
			BaseAddress: le.Uint64(data[off:]),
			RegionSize:  le.Uint64(data[off+24:]),
			State:       le.Uint32(data[off+32:]),
			Protect:     le.Uint32(data[off+36:]),
			Type:        le.Uint32(data[off+40:]),
		}
	}
	return infos
}

func parseSystemMemInfo(data []byte) *SystemMemoryInfo {
	// Revision(2) + Flags(2) + BasicInfo(0x34) + FileCacheInfo(0x3C) + BasicPerfInfo(0x20) + PerfInfo(0x158) = 0x1EC (492)
	if len(data) < 0x1EC {
		return nil
	}
	return &SystemMemoryInfo{
		PageSize:           le.Uint32(data[0x04+0x04:]),
		NumberOfPhysPages:  le.Uint32(data[0x04+0x08:]),
		NumberOfProcessors: le.Uint32(data[0x04+0x30:]),
		AvailablePages:     le.Uint64(data[0x74:]),
		CommittedPages:     le.Uint64(data[0x74+0x08:]),
		CommitLimit:        le.Uint64(data[0x74+0x10:]),
		PeakCommitment:     le.Uint64(data[0x74+0x18:]),
		PagedPoolPages:     le.Uint32(data[0x94+0x70:]),
		NonPagedPoolPages:  le.Uint32(data[0x94+0x74:]),
	}
}

func parseFunctionTables(data []byte) []FunctionTable {
	// Header: SizeOfHeader(4) + SizeOfDescriptor(4) + SizeOfNativeDescriptor(4) +
	//         SizeOfFunctionEntry(4) + NumberOfDescriptors(4) + SizeOfAlignPad(4) = 24
	if len(data) < 24 {
		return nil
	}
	headerSize := le.Uint32(data[0:])
	descSize := le.Uint32(data[4:])
	nativeSize := le.Uint32(data[8:])
	entrySize := le.Uint32(data[12:])
	count := le.Uint32(data[16:])
	alignPad := le.Uint32(data[20:])

	if descSize < 32 || entrySize < 12 || count == 0 {
		return nil
	}

	tables := make([]FunctionTable, 0, count)
	off := int(headerSize + alignPad)
	for i := uint32(0); i < count && off+int(descSize) <= len(data); i++ {
		ft := FunctionTable{
			MinAddress:  le.Uint64(data[off:]),
			MaxAddress:  le.Uint64(data[off+8:]),
			BaseAddress: le.Uint64(data[off+16:]),
		}
		entryCount := le.Uint32(data[off+24:])
		entryAlignPad := le.Uint32(data[off+28:])
		off += int(descSize) + int(nativeSize)

		if entrySize == 12 {
			for j := uint32(0); j < entryCount && off+12 <= len(data); j++ {
				ft.Entries = append(ft.Entries, FunctionEntry{
					BeginAddress:      le.Uint32(data[off:]),
					EndAddress:        le.Uint32(data[off+4:]),
					UnwindInfoAddress: le.Uint32(data[off+8:]),
				})
				off += int(entrySize)
			}
		} else {
			off += int(entryCount) * int(entrySize)
		}
		off += int(entryAlignPad)
		tables = append(tables, ft)
	}
	return tables
}

func parseVmCounters(data []byte) *ProcessVmCounters {
	// Revision 1: 0x50 (80) bytes. Revision 2: 0x98 (152) bytes.
	// Both share the same first 80 bytes of useful fields.
	if len(data) < 0x50 {
		return nil
	}
	return &ProcessVmCounters{
		PageFaultCount:             le.Uint32(data[0x04:]),
		PeakWorkingSetSize:         le.Uint64(data[0x08:]),
		WorkingSetSize:             le.Uint64(data[0x10:]),
		QuotaPeakPagedPoolUsage:    le.Uint64(data[0x18:]),
		QuotaPagedPoolUsage:        le.Uint64(data[0x20:]),
		QuotaPeakNonPagedPoolUsage: le.Uint64(data[0x28:]),
		QuotaNonPagedPoolUsage:     le.Uint64(data[0x30:]),
		PagefileUsage:              le.Uint64(data[0x38:]),
		PeakPagefileUsage:          le.Uint64(data[0x40:]),
		PrivateUsage:               le.Uint64(data[0x48:]),
	}
}

func parseBreakpadInfo(data []byte) *BreakpadInfo {
	if len(data) < 12 {
		return nil
	}
	return &BreakpadInfo{
		Validity:           le.Uint32(data[0:]),
		DumpThreadID:       le.Uint32(data[4:]),
		RequestingThreadID: le.Uint32(data[8:]),
	}
}

func parseAssertionInfo(data []byte) *AssertionInfo {
	if len(data) < 776 {
		return nil
	}
	return &AssertionInfo{
		Expression: decodeUTF16Fixed(data[0:256]),
		Function:   decodeUTF16Fixed(data[256:512]),
		File:       decodeUTF16Fixed(data[512:768]),
		Line:       le.Uint32(data[768:]),
		Type:       le.Uint32(data[772:]),
	}
}

func align8(n int) int {
	if r := n % 8; r != 0 {
		return n + 8 - r
	}
	return n
}

func parseStacktraces(data []byte) []Stacktrace {
	if len(data) < 16 {
		return nil
	}
	version := le.Uint32(data[0:])
	if version != 1 {
		return nil
	}
	numThreads := le.Uint32(data[4:])
	numFrames := le.Uint32(data[8:])
	symbolBytes := le.Uint32(data[12:])

	off := align8(16)

	threadSize := int(numThreads) * 12
	if off+threadSize > len(data) {
		return nil
	}
	type rawThread struct {
		id, start, count uint32
	}
	threads := make([]rawThread, numThreads)
	for i := range threads {
		threads[i] = rawThread{
			id:    le.Uint32(data[off:]),
			start: le.Uint32(data[off+4:]),
			count: le.Uint32(data[off+8:]),
		}
		off += 12
	}
	off = align8(off)

	frameSize := int(numFrames) * 16
	if off+frameSize > len(data) {
		return nil
	}
	type rawFrame struct {
		addr   uint64
		symOff uint32
		symLen uint32
	}
	frames := make([]rawFrame, numFrames)
	for i := range frames {
		frames[i] = rawFrame{
			addr:   le.Uint64(data[off:]),
			symOff: le.Uint32(data[off+8:]),
			symLen: le.Uint32(data[off+12:]),
		}
		off += 16
	}
	off = align8(off)

	if off+int(symbolBytes) > len(data) {
		return nil
	}
	symbols := data[off : off+int(symbolBytes)]

	result := make([]Stacktrace, len(threads))
	for i, t := range threads {
		st := Stacktrace{ThreadID: t.id}
		end := t.start + t.count
		if end > numFrames {
			end = numFrames
		}
		for j := t.start; j < end; j++ {
			f := frames[j]
			sym := ""
			if f.symLen > 0 && int(f.symOff+f.symLen) <= len(symbols) {
				sym = string(symbols[f.symOff : f.symOff+f.symLen])
			}
			st.Frames = append(st.Frames, StackFrame{
				InstructionAddr: f.addr,
				Symbol:          sym,
			})
		}
		result[i] = st
	}
	return result
}

func readMinidumpString(data []byte, rva uint32) string {
	off := int(rva)
	if off+4 > len(data) {
		return ""
	}
	length := le.Uint32(data[off:])
	off += 4
	if off+int(length) > len(data) {
		return ""
	}
	raw := data[off : off+int(length)]
	return decodeUTF16(raw)
}

func decodeUTF16(data []byte) string {
	u16 := make([]uint16, len(data)/2)
	for i := range u16 {
		u16[i] = le.Uint16(data[i*2:])
	}
	return strings.TrimRight(string(utf16.Decode(u16)), "\x00")
}

func decodeUTF16Fixed(data []byte) string {
	u16 := make([]uint16, len(data)/2)
	for i := range u16 {
		u16[i] = le.Uint16(data[i*2:])
	}
	s := string(utf16.Decode(u16))
	if idx := strings.IndexByte(s, 0); idx >= 0 {
		s = s[:idx]
	}
	return s
}

var le = binary.LittleEndian
