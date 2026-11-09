package fbhttp

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRedisServer is a lightweight, in-process TCP server implementing
// the subset of the RESP protocol needed for hermetic testing.
type mockRedisServer struct {
	listener    net.Listener
	mu          sync.RWMutex
	store       map[string]string
	expirations map[string]time.Time
	expireCalls map[string]int
	closed      bool
	conns       map[net.Conn]struct{}
}

func startMockRedisServer(t *testing.T) *mockRedisServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &mockRedisServer{
		listener:    ln,
		store:       make(map[string]string),
		expirations: make(map[string]time.Time),
		expireCalls: make(map[string]int),
		conns:       make(map[net.Conn]struct{}),
	}

	go s.acceptLoop()

	t.Cleanup(s.Close)
	return s
}

func (s *mockRedisServer) Addr() string {
	return s.listener.Addr().String()
}

func (s *mockRedisServer) URL() string {
	return "redis://" + s.Addr()
}

func (s *mockRedisServer) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	_ = s.listener.Close()
	for c := range s.conns {
		_ = c.Close()
	}
	s.mu.Unlock()
}

func (s *mockRedisServer) SetRaw(key, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = val
}

func (s *mockRedisServer) GetRaw(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.store[key]
	return v, ok
}

func (s *mockRedisServer) ExpireCalls(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.expireCalls[key]
}

func (s *mockRedisServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = conn.Close()
			return
		}
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		go s.handleConn(conn)
	}
}

func (s *mockRedisServer) handleConn(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		args, err := readRESPCommand(reader)
		if err != nil {
			return
		}
		if len(args) == 0 {
			continue
		}

		cmd := strings.ToUpper(args[0])
		switch cmd {
		case "PING":
			if len(args) > 1 {
				writeRESPBulkString(writer, args[1])
			} else {
				writeRESPSimpleString(writer, "PONG")
			}
		case "HELLO":
			// Return unknown command so go-redis falls back cleanly to RESP2
			writeRESPError(writer, "ERR unknown command 'HELLO'")
		case "CLIENT":
			// Handshake commands like CLIENT SETINFO
			writeRESPSimpleString(writer, "OK")
		case "GET":
			if len(args) < 2 {
				writeRESPError(writer, "ERR wrong number of arguments for 'get' command")
			} else {
				key := args[1]
				s.mu.RLock()
				val, ok := s.store[key]
				s.mu.RUnlock()
				if !ok {
					writeRESPNullBulkString(writer)
				} else {
					writeRESPBulkString(writer, val)
				}
			}
		case "SET":
			if len(args) < 3 {
				writeRESPError(writer, "ERR wrong number of arguments for 'set' command")
			} else {
				key := args[1]
				val := args[2]
				var ttl time.Duration
				for i := 3; i < len(args); i++ {
					opt := strings.ToUpper(args[i])
					if opt == "EX" && i+1 < len(args) {
						if secs, err := strconv.Atoi(args[i+1]); err == nil {
							ttl = time.Duration(secs) * time.Second
						}
						i++
					} else if opt == "PX" && i+1 < len(args) {
						if ms, err := strconv.Atoi(args[i+1]); err == nil {
							ttl = time.Duration(ms) * time.Millisecond
						}
						i++
					}
				}
				s.mu.Lock()
				s.store[key] = val
				if ttl > 0 {
					s.expirations[key] = time.Now().Add(ttl)
				}
				s.mu.Unlock()
				writeRESPSimpleString(writer, "OK")
			}
		case "EXPIRE":
			if len(args) < 3 {
				writeRESPError(writer, "ERR wrong number of arguments for 'expire' command")
			} else {
				key := args[1]
				secs, _ := strconv.Atoi(args[2])
				s.mu.Lock()
				s.expireCalls[key]++
				_, exists := s.store[key]
				if exists {
					s.expirations[key] = time.Now().Add(time.Duration(secs) * time.Second)
				}
				s.mu.Unlock()
				if exists {
					writeRESPInteger(writer, 1)
				} else {
					writeRESPInteger(writer, 0)
				}
			}
		case "DEL":
			if len(args) < 2 {
				writeRESPError(writer, "ERR wrong number of arguments for 'del' command")
			} else {
				deleted := 0
				s.mu.Lock()
				for _, k := range args[1:] {
					if _, exists := s.store[k]; exists {
						delete(s.store, k)
						delete(s.expirations, k)
						deleted++
					}
				}
				s.mu.Unlock()
				writeRESPInteger(writer, int64(deleted))
			}
		case "QUIT":
			writeRESPSimpleString(writer, "OK")
			_ = writer.Flush()
			return
		default:
			writeRESPError(writer, fmt.Sprintf("ERR unknown command '%s'", cmd))
		}

		if err := writer.Flush(); err != nil {
			return
		}
	}
}

func readRESPCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return nil, nil
	}

	if line[0] == '*' {
		count, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		args := make([]string, 0, count)
		for i := 0; i < count; i++ {
			header, err := r.ReadString('\n')
			if err != nil {
				return nil, err
			}
			header = strings.TrimRight(header, "\r\n")
			if len(header) == 0 || header[0] != '$' {
				return nil, fmt.Errorf("expected bulk string '$', got %q", header)
			}
			argLen, err := strconv.Atoi(header[1:])
			if err != nil {
				return nil, err
			}
			buf := make([]byte, argLen+2) // data + \r\n
			if _, err := io.ReadFull(r, buf); err != nil {
				return nil, err
			}
			args = append(args, string(buf[:argLen]))
		}
		return args, nil
	}

	// Inline command (fallback)
	return strings.Fields(line), nil
}

func writeRESPSimpleString(w *bufio.Writer, s string) {
	_, _ = fmt.Fprintf(w, "+%s\r\n", s)
}

func writeRESPError(w *bufio.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "-%s\r\n", msg)
}

func writeRESPInteger(w *bufio.Writer, n int64) {
	_, _ = fmt.Fprintf(w, ":%d\r\n", n)
}

func writeRESPBulkString(w *bufio.Writer, s string) {
	_, _ = fmt.Fprintf(w, "$%d\r\n%s\r\n", len(s), s)
}

func writeRESPNullBulkString(w *bufio.Writer) {
	_, _ = fmt.Fprintf(w, "$-1\r\n")
}

// Test cases:

func TestRedisUploadCache_RegisterWritesNyxPrefix(t *testing.T) {
	server := startMockRedisServer(t)
	cache, err := newRedisUploadCache(server.URL())
	require.NoError(t, err)
	t.Cleanup(cache.Close)

	filePath := "/srv/storage/report.pdf"
	fileSize := int64(1048576)

	cache.Register(filePath, fileSize, nil)

	// Verify key with canonical nyx:upload: prefix exists
	val, ok := server.GetRaw("nyx:upload:" + filePath)
	assert.True(t, ok, "canonical nyx:upload: key should exist")
	assert.Equal(t, strconv.FormatInt(fileSize, 10), val)

	// Verify legacy key was NOT created
	_, legacyOk := server.GetRaw("filebrowser:upload:" + filePath)
	assert.False(t, legacyOk, "legacy filebrowser:upload: key should not be created on Register")
}

func TestRedisUploadCache_GetLength_ReadsNyxKey(t *testing.T) {
	server := startMockRedisServer(t)
	cache, err := newRedisUploadCache(server.URL())
	require.NoError(t, err)
	t.Cleanup(cache.Close)

	filePath := "/srv/storage/archive.zip"
	expectedSize := int64(5242880)

	// Seed canonical key
	server.SetRaw("nyx:upload:"+filePath, strconv.FormatInt(expectedSize, 10))

	size, err := cache.GetLength(filePath)
	require.NoError(t, err)
	assert.Equal(t, expectedSize, size)

	// GetLength calls Touch, so EXPIRE should have been called on canonical key
	assert.GreaterOrEqual(t, server.ExpireCalls("nyx:upload:"+filePath), 1)
}

func TestRedisUploadCache_GetLength_FallsBackToLegacyKey(t *testing.T) {
	server := startMockRedisServer(t)
	cache, err := newRedisUploadCache(server.URL())
	require.NoError(t, err)
	t.Cleanup(cache.Close)

	filePath := "/srv/storage/legacy_upload.iso"
	legacySize := int64(4294967296)

	// Seed ONLY legacy key (simulating an in-flight upload from prior release)
	server.SetRaw("filebrowser:upload:"+filePath, strconv.FormatInt(legacySize, 10))

	size, err := cache.GetLength(filePath)
	require.NoError(t, err)
	assert.Equal(t, legacySize, size)

	// Verify legacy key is preserved and NOT deleted
	val, ok := server.GetRaw("filebrowser:upload:" + filePath)
	assert.True(t, ok, "legacy key must be preserved during read fallback")
	assert.Equal(t, strconv.FormatInt(legacySize, 10), val)

	// Verify Touch refreshed the legacy key's TTL since Nyx key was absent
	assert.GreaterOrEqual(t, server.ExpireCalls("filebrowser:upload:"+filePath), 1)
}

func TestRedisUploadCache_Touch_RefreshesTTL(t *testing.T) {
	server := startMockRedisServer(t)
	cache, err := newRedisUploadCache(server.URL())
	require.NoError(t, err)
	t.Cleanup(cache.Close)

	t.Run("touches nyx key when present", func(t *testing.T) {
		path := "/srv/storage/active.mp4"
		server.SetRaw("nyx:upload:"+path, "123456")

		cache.Touch(path)
		assert.Equal(t, 1, server.ExpireCalls("nyx:upload:"+path))
		assert.Equal(t, 0, server.ExpireCalls("filebrowser:upload:"+path))
	})

	t.Run("touches legacy key when nyx key is absent", func(t *testing.T) {
		path := "/srv/storage/legacy.mp4"
		server.SetRaw("filebrowser:upload:"+path, "654321")

		cache.Touch(path)
		// nyx key was checked via Expire, returned false (0 in mock), then legacy key was touched
		assert.Equal(t, 1, server.ExpireCalls("nyx:upload:"+path))
		assert.Equal(t, 1, server.ExpireCalls("filebrowser:upload:"+path))
	})
}

func TestRedisUploadCache_Complete_CleansUpBothKeys(t *testing.T) {
	server := startMockRedisServer(t)
	cache, err := newRedisUploadCache(server.URL())
	require.NoError(t, err)
	t.Cleanup(cache.Close)

	filePath := "/srv/storage/both_keys.tar"
	server.SetRaw("nyx:upload:"+filePath, "1000")
	server.SetRaw("filebrowser:upload:"+filePath, "1000")

	cache.Complete(filePath)

	_, nyxOk := server.GetRaw("nyx:upload:" + filePath)
	assert.False(t, nyxOk, "nyx key should be deleted")

	_, legacyOk := server.GetRaw("filebrowser:upload:" + filePath)
	assert.False(t, legacyOk, "legacy key should be deleted")
}

func TestRedisUploadCache_NonExistentKeyReturnsError(t *testing.T) {
	server := startMockRedisServer(t)
	cache, err := newRedisUploadCache(server.URL())
	require.NoError(t, err)
	t.Cleanup(cache.Close)

	size, err := cache.GetLength("/srv/storage/does_not_exist.bin")
	assert.Error(t, err)
	assert.Equal(t, int64(0), size)
	assert.Contains(t, err.Error(), "no active upload found for the given path")
}

func TestRedisUploadCache_InvalidLengthReturnsError(t *testing.T) {
	server := startMockRedisServer(t)
	cache, err := newRedisUploadCache(server.URL())
	require.NoError(t, err)
	t.Cleanup(cache.Close)

	filePath := "/srv/storage/corrupted.bin"
	server.SetRaw("nyx:upload:"+filePath, "not-a-number")

	size, err := cache.GetLength(filePath)
	assert.Error(t, err)
	assert.Equal(t, int64(0), size)
	assert.Contains(t, err.Error(), "invalid upload length in cache")
}

func TestRedisUploadCache_NewUploadCache(t *testing.T) {
	server := startMockRedisServer(t)

	// Testing constructor via NewUploadCache
	c, err := NewUploadCache(server.URL())
	require.NoError(t, err)
	require.NotNil(t, c)
	t.Cleanup(c.Close)

	_, isRedis := c.(*redisUploadCache)
	assert.True(t, isRedis, "NewUploadCache with URL should produce *redisUploadCache")

	// Empty URL should produce memory cache
	memCache, err := NewUploadCache("")
	require.NoError(t, err)
	require.NotNil(t, memCache)
	t.Cleanup(memCache.Close)
	_, isMem := memCache.(*memoryUploadCache)
	assert.True(t, isMem, "NewUploadCache with empty string should produce *memoryUploadCache")

	// Invalid URL should error
	_, err = NewUploadCache("invalid-url-with-no-scheme")
	assert.Error(t, err)

	// Empty redisURL in newRedisUploadCache
	_, err = newRedisUploadCache("")
	assert.Error(t, err)
}

func TestRedisUploadCache_Concurrency(t *testing.T) {
	server := startMockRedisServer(t)
	cache, err := newRedisUploadCache(server.URL())
	require.NoError(t, err)
	t.Cleanup(cache.Close)

	const numGoroutines = 25
	const opsPerGoroutine = 40

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				path := fmt.Sprintf("/srv/data/worker_%d_file_%d.dat", workerID, j)
				size := int64(1024 + workerID*100 + j)

				// 1. Register
				cache.Register(path, size, nil)

				// 2. Touch
				cache.Touch(path)

				// 3. GetLength
				gotSize, err := cache.GetLength(path)
				if err != nil {
					t.Errorf("worker %d failed GetLength for %s: %v", workerID, path, err)
					return
				}
				if gotSize != size {
					t.Errorf("worker %d got size %d, expected %d", workerID, gotSize, size)
					return
				}

				// 4. Complete
				cache.Complete(path)

				// 5. Verify deleted
				_, err = cache.GetLength(path)
				if err == nil {
					t.Errorf("worker %d expected error after Complete for %s", workerID, path)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}
