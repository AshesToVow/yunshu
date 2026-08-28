package dbconn

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/mysqlbackup"

	"crypto/cipher"

	mysqldriver "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/ssh"
)

// OpenParams 打开业务库连接所需参数。
type OpenParams struct {
	Driver      string
	Host        string
	Port        int
	Database    string
	Username    string
	Password    string
	SSLMode     string
	ConnectMode string
	ServerID    *uint
}

// Session 持有可选 SSH 隧道与 database/sql 连接。
type Session struct {
	DB     *sql.DB
	Tunnel *SSHTunnel
	SSH    *ssh.Client
}

func (s *Session) Close() {
	if s == nil {
		return
	}
	if s.DB != nil {
		_ = s.DB.Close()
	}
	if s.Tunnel != nil {
		_ = s.Tunnel.Close()
	}
	if s.SSH != nil {
		_ = s.SSH.Close()
	}
}

// SSHDialer 通过 CMDB Server SSH 建立客户端。
type SSHDialer interface {
	DialServer(ctx context.Context, serverID uint) (*ssh.Client, error)
}

// OpenSession 打开连接会话（含 SSH 隧道时一并管理生命周期）。
func OpenSession(ctx context.Context, p OpenParams, dialer SSHDialer) (*Session, error) {
	mode := strings.ToLower(strings.TrimSpace(p.ConnectMode))
	if mode == model.DbConnectSSHTunnel && p.ServerID != nil && *p.ServerID > 0 {
		if dialer == nil {
			return nil, fmt.Errorf("ssh dialer required for tunnel mode")
		}
		cli, err := dialer.DialServer(ctx, *p.ServerID)
		if err != nil {
			return nil, err
		}
		remoteHost := strings.TrimSpace(p.Host)
		if remoteHost == "" {
			remoteHost = "127.0.0.1"
		}
		remotePort := p.Port
		if remotePort <= 0 {
			if strings.ToLower(p.Driver) == model.DbDriverPostgres {
				remotePort = 5432
			} else {
				remotePort = 3306
			}
		}
		tunnel, err := StartSSHTunnel(cli, "127.0.0.1", 0, remoteHost, remotePort)
		if err != nil {
			cli.Close()
			return nil, err
		}
		tunHost, tunPort := tunnel.LocalHost, tunnel.LocalPort
		tp := p
		tp.ConnectMode = model.DbConnectDirect
		tp.Host = tunHost
		tp.Port = tunPort
		db, err := openDB(ctx, tp)
		if err != nil {
			tunnel.Close()
			cli.Close()
			return nil, err
		}
		return &Session{DB: db, Tunnel: tunnel, SSH: cli}, nil
	}
	db, err := openDB(ctx, p)
	if err != nil {
		return nil, err
	}
	return &Session{DB: db}, nil
}

func openDB(ctx context.Context, p OpenParams) (*sql.DB, error) {
	driver := strings.ToLower(strings.TrimSpace(p.Driver))
	switch driver {
	case model.DbDriverMySQL, "":
		return openMySQL(p)
	case model.DbDriverPostgres:
		return openPostgres(p)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", p.Driver)
	}
}

func openMySQL(p OpenParams) (*sql.DB, error) {
	host := strings.TrimSpace(p.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	port := p.Port
	if port <= 0 {
		port = 3306
	}
	dbName := strings.TrimSpace(p.Database)
	if dbName == "" {
		dbName = "mysql"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s&readTimeout=30s&writeTimeout=30s&multiStatements=false",
		p.Username, p.Password, host, port, dbName)
	switch strings.ToLower(strings.TrimSpace(p.SSLMode)) {
	case "preferred":
		dsn += "&tls=preferred"
	case "required", "require", "true":
		dsn += "&tls=true"
	case "verify_ca", "verify-ca":
		dsn += "&tls=skip-verify"
	case "verify_identity", "verify-full":
		dsn += "&tls=preferred"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(2 * time.Minute)
	return db, nil
}

func openPostgres(p OpenParams) (*sql.DB, error) {
	host := strings.TrimSpace(p.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	port := p.Port
	if port <= 0 {
		port = 5432
	}
	dbName := strings.TrimSpace(p.Database)
	if dbName == "" {
		dbName = "postgres"
	}
	sslmode := strings.TrimSpace(p.SSLMode)
	if sslmode == "" {
		sslmode = "disable"
	}
	u := buildPostgresURL(host, port, p.Username, p.Password, dbName, sslmode)
	db, err := sql.Open("pgx", u)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(2 * time.Minute)
	return db, nil
}

func buildPostgresURL(host string, port int, user, password, database, sslmode string) string {
	u := &url.URL{Scheme: "postgres", Host: fmt.Sprintf("%s:%d", host, port), Path: database}
	if password != "" {
		u.User = url.UserPassword(user, password)
	} else {
		u.User = url.User(user)
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	q.Set("connect_timeout", "5")
	u.RawQuery = q.Encode()
	return u.String()
}

// Ping 探活。
func Ping(ctx context.Context, p OpenParams, dialer SSHDialer) error {
	sess, err := OpenSession(ctx, p, dialer)
	if err != nil {
		return err
	}
	defer sess.Close()
	pctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return sess.DB.PingContext(pctx)
}

// PingDirect 无 SSH 的直连探活（供快速检测）。
func PingDirect(ctx context.Context, p OpenParams) error {
	switch strings.ToLower(strings.TrimSpace(p.Driver)) {
	case model.DbDriverPostgres:
		return mysqlbackup.PingPostgres(ctx, p.Host, p.Port, p.Username, p.Password, p.Database, p.SSLMode)
	default:
		return mysqlbackup.Ping(ctx, p.Host, p.Port, p.Username, p.Password, "")
	}
}

// DecryptPassword 解密实例密码。
func DecryptPassword(aead cipher.AEAD, enc string) (string, error) {
	if strings.TrimSpace(enc) == "" {
		return "", nil
	}
	return decryptString(aead, enc)
}

var decryptString = func(aead cipher.AEAD, enc string) (string, error) {
	return "", fmt.Errorf("decrypt not configured")
}

// SetDecryptFunc 由 service 初始化时注入。
func SetDecryptFunc(fn func(cipher.AEAD, string) (string, error)) {
	if fn != nil {
		decryptString = fn
	}
}

// 避免未使用导入
var _ = mysqldriver.ErrInvalidConn

// SSHTunnel 本地 TCP 监听转发到远端目标。
type SSHTunnel struct {
	SSHClient  *ssh.Client
	LocalHost  string
	LocalPort  int
	RemoteHost string
	RemotePort int

	listener net.Listener
	wg       sync.WaitGroup
	closed   chan struct{}
}

// StartSSHTunnel 在本地监听并转发到远端 host:port。
func StartSSHTunnel(sshClient *ssh.Client, localHost string, localPort int, remoteHost string, remotePort int) (*SSHTunnel, error) {
	if sshClient == nil {
		return nil, fmt.Errorf("ssh client required")
	}
	if localHost == "" {
		localHost = "127.0.0.1"
	}
	if localPort <= 0 {
		ln, err := net.Listen("tcp", localHost+":0")
		if err != nil {
			return nil, err
		}
		addr := ln.Addr().(*net.TCPAddr)
		localPort = addr.Port
		_ = ln.Close()
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", localHost, localPort))
	if err != nil {
		return nil, err
	}
	t := &SSHTunnel{
		SSHClient:  sshClient,
		LocalHost:  localHost,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		listener:   ln,
		closed:     make(chan struct{}),
	}
	t.wg.Go(t.acceptLoop)
	return t, nil
}

func (t *SSHTunnel) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.closed:
				return
			default:
			}
			return
		}
		t.wg.Go(func() { t.handleConn(conn) })
	}
}

func (t *SSHTunnel) handleConn(local net.Conn) {
	defer local.Close()
	remote, err := t.SSHClient.Dial("tcp", fmt.Sprintf("%s:%d", t.RemoteHost, t.RemotePort))
	if err != nil {
		return
	}
	defer remote.Close()
	done := make(chan struct{}, 2)
	go copyAndClose(done, local, remote)
	go copyAndClose(done, remote, local)
	<-done
}

func copyAndClose(done chan struct{}, dst io.ReadWriter, src io.Reader) {
	_, _ = io.Copy(dst, src)
	select {
	case done <- struct{}{}:
	default:
	}
}

// Close 关闭隧道。
func (t *SSHTunnel) Close() error {
	close(t.closed)
	if t.listener != nil {
		_ = t.listener.Close()
	}
	waitDone := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
	}
	return nil
}
