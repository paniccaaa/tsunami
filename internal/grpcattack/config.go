package grpcattack

import "time"

type Config struct {
	Target    string
	Service   string
	Method    string
	ProtoFile string
	Data      string
	Metadata  []string
	Insecure  bool
	CACert    string

	Duration    time.Duration
	Timeout     time.Duration
	Workers     uint
	Connections uint
	RPS         int
	Output      string
}
