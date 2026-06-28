package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/jpnurmi/slope/envelope"
)

func parseEnvelopeFile(path string) (*envelope.Envelope, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return envelope.Parse(f)
}

func cmdHeader(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: slope header <file> [index]")
	}
	env, err := parseEnvelopeFile(args[0])
	if err != nil {
		return err
	}
	if len(args) == 2 {
		idx, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid index: %s", args[1])
		}
		if idx < 0 || idx > len(env.Items) {
			return fmt.Errorf("index out of range: %d (envelope has %d items)", idx, len(env.Items))
		}
		var raw json.RawMessage
		if idx == 0 {
			raw = env.Header
		} else {
			raw = env.Items[idx-1].Header
		}
		compact, err := json.Marshal(raw)
		if err != nil {
			return fmt.Errorf("compacting header: %w", err)
		}
		fmt.Printf("%s\n", compact)
		return nil
	}
	compact, err := json.Marshal(json.RawMessage(env.Header))
	if err != nil {
		return fmt.Errorf("compacting header: %w", err)
	}
	fmt.Printf("0: %s\n", compact)
	for i, item := range env.Items {
		compact, err := json.Marshal(json.RawMessage(item.Header))
		if err != nil {
			return fmt.Errorf("compacting header %d: %w", i+1, err)
		}
		fmt.Printf("%d: %s\n", i+1, compact)
	}
	return nil
}

func cmdPayload(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: slope payload <file> [index]")
	}
	env, err := parseEnvelopeFile(args[0])
	if err != nil {
		return err
	}
	if len(args) == 1 {
		totalSize := 0
		for _, item := range env.Items {
			totalSize += len(item.Payload)
		}
		fmt.Printf("0: <%d items, %d bytes>\n", len(env.Items), totalSize)
		for i, item := range env.Items {
			if envelope.IsBinary(item.Payload) {
				fmt.Printf("%d: <binary %d bytes>\n", i+1, len(item.Payload))
			} else {
				compact, err := json.Marshal(json.RawMessage(item.Payload))
				if err != nil {
					fmt.Printf("%d: <text %d bytes>\n", i+1, len(item.Payload))
				} else {
					fmt.Printf("%d: %s\n", i+1, compact)
				}
			}
		}
		return nil
	}
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid index: %s", args[1])
	}
	if idx < 0 || idx > len(env.Items) {
		return fmt.Errorf("index out of range: %d (envelope has %d items)", idx, len(env.Items))
	}
	if idx == 0 {
		return env.SerializeItems(os.Stdout)
	}
	_, err = os.Stdout.Write(env.Items[idx-1].Payload)
	return err
}
