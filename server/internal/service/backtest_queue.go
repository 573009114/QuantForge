package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BacktestQueue interface {
	Enqueue(priority int, item queuedBacktest) error
	Dequeue(ctx context.Context) (queuedBacktest, bool, error)
}

type memoryBacktestQueue struct {
	mu      sync.Mutex
	pending map[int][]queuedBacktest
	notify  chan struct{}
}

func newMemoryBacktestQueue() *memoryBacktestQueue {
	return &memoryBacktestQueue{pending: map[int][]queuedBacktest{}, notify: make(chan struct{}, 1)}
}
func (q *memoryBacktestQueue) Enqueue(priority int, item queuedBacktest) error {
	q.mu.Lock()
	q.pending[priority] = append(q.pending[priority], item)
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return nil
}
func (q *memoryBacktestQueue) Dequeue(ctx context.Context) (queuedBacktest, bool, error) {
	for {
		q.mu.Lock()
		for p := 10; p >= 1; p-- {
			arr := q.pending[p]
			if len(arr) > 0 {
				it := arr[0]
				q.pending[p] = arr[1:]
				q.mu.Unlock()
				return it, true, nil
			}
		}
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return queuedBacktest{}, false, ctx.Err()
		case <-q.notify:
		}
	}
}

type redisBacktestQueue struct {
	addr string
	key  string
}

func newRedisBacktestQueueFromEnv() (BacktestQueue, bool) {
	addr := strings.TrimSpace(os.Getenv("QF_REDIS_ADDR"))
	if addr == "" {
		return nil, false
	}
	key := os.Getenv("QF_REDIS_BACKTEST_QUEUE_KEY")
	if key == "" {
		key = "qf:backtest:queue"
	}
	return &redisBacktestQueue{addr: addr, key: key}, true
}

func (q *redisBacktestQueue) Enqueue(priority int, item queuedBacktest) error {
	payload := item.tenantID + "|" + item.jobID
	_, err := q.cmd("LPUSH", q.key+":p"+strconv.Itoa(priority), payload)
	return err
}

func (q *redisBacktestQueue) Dequeue(ctx context.Context) (queuedBacktest, bool, error) {
	args := []string{"BRPOP"}
	for p := 10; p >= 1; p-- {
		args = append(args, q.key+":p"+strconv.Itoa(p))
	}
	args = append(args, "1")
	for {
		select {
		case <-ctx.Done():
			return queuedBacktest{}, false, ctx.Err()
		default:
		}
		resp, err := q.cmd(args...)
		if err != nil {
			return queuedBacktest{}, false, err
		}
		arr, ok := resp.([]any)
		if !ok || len(arr) < 2 {
			continue
		}
		payload, _ := arr[1].(string)
		parts := strings.SplitN(payload, "|", 2)
		if len(parts) != 2 {
			continue
		}
		return queuedBacktest{tenantID: parts[0], jobID: parts[1]}, true, nil
	}
}

func (q *redisBacktestQueue) cmd(args ...string) (any, error) {
	conn, err := net.DialTimeout("tcp", q.addr, 800*time.Millisecond)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	var b bytes.Buffer
	fmt.Fprintf(&b, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(a), a)
	}
	if _, err = conn.Write(b.Bytes()); err != nil {
		return nil, err
	}
	return readRESP(bufio.NewReader(conn))
}

func readRESP(r *bufio.Reader) (any, error) {
	t, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	switch t {
	case '+':
		return line, nil
	case '-':
		return nil, errors.New(line)
	case ':':
		n, _ := strconv.ParseInt(line, 10, 64)
		return n, nil
	case '$':
		n, _ := strconv.Atoi(line)
		if n < 0 {
			return "", nil
		}
		buf := make([]byte, n+2)
		if _, err = r.Read(buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	case '*':
		n, _ := strconv.Atoi(line)
		if n < 0 {
			return []any{}, nil
		}
		arr := make([]any, 0, n)
		for i := 0; i < n; i++ {
			v, e := readRESP(r)
			if e != nil {
				return nil, e
			}
			arr = append(arr, v)
		}
		return arr, nil
	default:
		return nil, errors.New("unknown resp type")
	}
}
