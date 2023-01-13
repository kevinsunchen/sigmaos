package perf

type Tselector string

// Suffixes
const (
	PPROF     Tselector = "_PPROF"
	PPROF_MEM           = "_PPROF_MEM"
	CPU                 = "_CPU"
	TPT                 = "_TPT"
)

// Tests & benchmarking
const (
	TEST  Tselector = "TEST"
	BENCH           = "BENCH"
)

// kernel procs
const (
	NAMED  Tselector = "NAMED"
	PROCD            = "PROCD"
	S3               = "S3"
	SCHEDD           = "SCHEDD"
)

// libs
const (
	GROUP Tselector = "GROUP"
)

// mr
const (
	MRMAPPER  Tselector = "MRMAPPER"
	MRREDUCER           = "MRREDUCER"
	SEQGREP             = "SEQGREP"
	SEQWC               = "SEQWC"
)

// kv
const (
	KVCLERK Tselector = "KVCLERK"
)

// hotel
const (
	HOTEL_WWW     Tselector = "HOTEL_WWW"
	HOTEL_GEO               = "HOTEL_GEO"
	HOTEL_RESERVE           = "HOTEL_RESERVE"
	HOTEL_SEARCH            = "HOTEL_SEARCH"
)

// cache
const (
	CACHECLERK Tselector = "CACHECLERK"
)

// microbenchmarks
const (
	WRITER     Tselector = "writer"
	BUFWRITER            = "bufwriter"
	ABUFWRITER           = "abufwriter"
	READER               = "reader"
	BUFREADER            = "bufreader"
	ABUFREADER           = "abufreader"
)