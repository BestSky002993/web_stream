package cache

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

var ErrMiss = errors.New("cache miss")

type Config struct {
	Enabled      bool
	Addr         string
	Password     string
	DB           int
	Prefix       string
	TTL          time.Duration
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	TLS          bool
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	if cfg.Prefix == "" {
		cfg.Prefix = "web_stream"
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 5 * time.Minute
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 500 * time.Millisecond
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 500 * time.Millisecond
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 500 * time.Millisecond
	}
	return &Client{cfg: cfg}
}

func (c *Client) Enabled() bool { return c != nil && c.cfg.Enabled && c.cfg.Addr != "" }

func (c *Client) GetJSON(ctx context.Context, key string, dest any) bool {
	b, err := c.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, ErrMiss) {
			log.Printf("[CACHE] event=miss key=%s reason=%v", key, err)
		} else {
			log.Printf("[CACHE] event=miss key=%s", key)
		}
		return false
	}
	if err := json.Unmarshal(b, dest); err != nil {
		log.Printf("[CACHE] event=miss key=%s reason=json_decode err=%v", key, err)
		return false
	}
	log.Printf("[CACHE] event=hit key=%s", key)
	return true
}

func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	b, err := json.Marshal(value)
	if err != nil {
		log.Printf("[CACHE] event=set_failed key=%s reason=json_encode err=%v", key, err)
		return
	}
	if err := c.Set(ctx, key, b, ttl); err != nil {
		log.Printf("[CACHE] event=set_failed key=%s err=%v", key, err)
		return
	}
	log.Printf("[CACHE] event=set key=%s ttl=%s", key, ttl)
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	if !c.Enabled() {
		return nil, ErrMiss
	}
	resp, err := c.do(ctx, "GET", c.key(key))
	if err != nil {
		return nil, err
	}
	if resp.nil {
		return nil, ErrMiss
	}
	if resp.kind != '$' {
		return nil, fmt.Errorf("unexpected redis GET response %q", resp.kind)
	}
	return resp.bulk, nil
}

func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if !c.Enabled() {
		return nil
	}
	if ttl <= 0 {
		ttl = c.cfg.TTL
	}
	_, err := c.do(ctx, "SET", c.key(key), string(value), "EX", strconv.FormatInt(int64(ttl.Seconds()), 10))
	return err
}

func (c *Client) Delete(ctx context.Context, keys ...string) {
	if !c.Enabled() || len(keys) == 0 {
		return
	}
	args := make([]string, 0, len(keys)+1)
	args = append(args, "DEL")
	for _, k := range keys {
		args = append(args, c.key(k))
	}
	if _, err := c.do(ctx, args...); err != nil {
		log.Printf("[CACHE] event=delete_failed keys=%d err=%v", len(keys), err)
	}
}

func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return ErrMiss
	}
	_, err := c.do(ctx, "PING")
	return err
}

func (c *Client) key(key string) string {
	return strings.TrimRight(c.cfg.Prefix, ":") + ":" + key
}

func (c *Client) do(ctx context.Context, args ...string) (redisResp, error) {
	var zero redisResp
	var d net.Dialer
	d.Timeout = c.cfg.DialTimeout
	conn, err := d.DialContext(ctx, "tcp", c.cfg.Addr)
	if err != nil {
		return zero, err
	}
	defer conn.Close()
	if c.cfg.TLS {
		tlsConn := tls.Client(conn, &tls.Config{MinVersion: tls.VersionTLS12})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return zero, err
		}
		conn = tlsConn
	}
	deadline := time.Now().Add(c.cfg.ReadTimeout + c.cfg.WriteTimeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	_ = conn.SetDeadline(deadline)
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	if c.cfg.Password != "" {
		if err := writeCommand(rw, "AUTH", c.cfg.Password); err != nil {
			return zero, err
		}
		if _, err := readResp(rw.Reader); err != nil {
			return zero, err
		}
	}
	if c.cfg.DB > 0 {
		if err := writeCommand(rw, "SELECT", strconv.Itoa(c.cfg.DB)); err != nil {
			return zero, err
		}
		if _, err := readResp(rw.Reader); err != nil {
			return zero, err
		}
	}
	if err := writeCommand(rw, args...); err != nil {
		return zero, err
	}
	return readResp(rw.Reader)
}

type redisResp struct {
	kind byte
	bulk []byte
	nil  bool
}

func writeCommand(rw *bufio.ReadWriter, args ...string) error {
	if _, err := fmt.Fprintf(rw, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(rw, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return rw.Flush()
}

func readResp(r *bufio.Reader) (redisResp, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return redisResp{}, err
	}
	switch prefix {
	case '+':
		_, err := r.ReadString('\n')
		return redisResp{kind: prefix}, err
	case '-':
		line, _ := r.ReadString('\n')
		return redisResp{}, errors.New(strings.TrimSpace(line))
	case ':':
		_, err := r.ReadString('\n')
		return redisResp{kind: prefix}, err
	case '$':
		line, err := r.ReadString('\n')
		if err != nil {
			return redisResp{}, err
		}
		n, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			return redisResp{}, err
		}
		if n < 0 {
			return redisResp{kind: prefix, nil: true}, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return redisResp{}, err
		}
		return redisResp{kind: prefix, bulk: buf[:n]}, nil
	default:
		return redisResp{}, fmt.Errorf("unknown redis response %q", prefix)
	}
}
