package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/nuriland/a2kit"
	"github.com/nuriland/a2kit/game"
)

// Five actors, 25 distinct skills each: decode, account and publish on the
// requested cadence, including the real bounded decoder/publisher queues.
func BenchmarkCombatPipeline(b *testing.B) {
	const count = 5000
	var log bytes.Buffer
	log.WriteString("{\"schema\":\"a2log/v0.1\",\"decoder\":\"benchmark\",\"source\":{\"kind\":\"feed\"},\"t0\":\"2026-10-03T12:00:00Z\"}\n")
	for n := 0; n < count; n++ {
		p := binary.AppendUvarint(nil, 7)
		p = binary.AppendUvarint(p, 4)
		p = binary.AppendUvarint(p, 0)
		p = binary.AppendUvarint(p, uint64(n%5+1))
		p = binary.LittleEndian.AppendUint32(p, uint32(1000000000+n/5%25))
		p = append(p, 1, 2, 0, 0, 0, 0)
		p = binary.LittleEndian.AppendUint32(p, 1)
		p = binary.AppendUvarint(p, 10000)
		p = binary.AppendUvarint(p, 5000)
		p = append(p, 1, 0)
		fmt.Fprintf(&log, "{\"t\":%d,\"opcode\":\"04 38\",\"flags\":[\"server\"],\"src\":\"10.0.0.2:13328\",\"dst\":\"10.0.0.1:10000\",\"payload\":\"%s\"}\n", n, base64.StdEncoding.EncodeToString(p))
	}
	fixture := log.String()
	for _, interval := range []int{50, 200, 1000} {
		b.Run(fmt.Sprintf("%dms", interval), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				r, err := a2kit.OpenReader(strings.NewReader(fixture), a2kit.Config{})
				if err != nil {
					b.Fatal(err)
				}
				messages := make(chan a2kit.Message, 256)
				errs := make(chan error, 1)
				go func() {
					defer close(messages)
					for msg, err := range r.Messages() {
						if err != nil {
							errs <- err
							return
						}
						messages <- msg
					}
				}()
				m := meter{}
				publisher := newSnapshotPublisher(io.Discard)
				seen := 0
				for msg := range messages {
					if hit, ok := msg.Event.(game.Hit); ok {
						m.add(msg.Time, hit)
						seen++
					}
					if seen%interval == 0 {
						publisher.offer(m.snapshot(identities{}))
					}
				}
				if err := publisher.close(); err != nil {
					b.Fatal(err)
				}
				r.Close()
				select {
				case err := <-errs:
					b.Fatal(err)
				default:
				}
				if seen != count {
					b.Fatalf("decoded %d/%d hits", seen, count)
				}
			}
			b.ReportMetric(float64(count*b.N)/b.Elapsed().Seconds(), "hits/s")
		})
	}
}

func BenchmarkFullPartySnapshot(b *testing.B) {
	m := meter{}
	for actor := game.Entity(1); actor <= 5; actor++ {
		for skill := game.Skill(1000000000); skill < 1000000025; skill++ {
			m.add(time.Unix(100, 0), game.Hit{Actor: actor, Skill: skill, Damage: 1000, Type: 3})
		}
	}
	encoder := json.NewEncoder(io.Discard)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if err := encoder.Encode(m.snapshot(identities{})); err != nil {
			b.Fatal(err)
		}
	}
}
