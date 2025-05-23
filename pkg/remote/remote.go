package remote

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/optiflow-os/tracelens-cli/pkg/file"
	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"
	"github.com/optiflow-os/tracelens-cli/pkg/log"

	"github.com/kevinburke/ssh_config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	defaultTimeoutSecs = 20
	// maxFileSize defines the max file size to download from remote. Default is 512Kb.
	maxFileSize = 512000
	defaultPort = 22
)

// Client communicates using sftp protocol.
type Client struct {
	User         string
	Pass         string
	HostKeyAlias string
	OriginalHost string
	Host         string
	Port         int
	Path         string
}

// WithDetection initializes and returns a heartbeat handle option, which
// can be used in a heartbeat processing pipeline to detect remote file and
// download to a temporary directory.
func WithDetection() heartbeat.HandleOption {
	return func(next heartbeat.Handle) heartbeat.Handle {
		return func(ctx context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			logger := log.Extract(ctx)
			logger.Debugln("execute remote file detection")

			var filtered []heartbeat.Heartbeat

			for _, h := range hh {
				if !h.IsRemote() {
					filtered = append(filtered, h)
					continue
				}

				tmpFile, err := os.CreateTemp("", fmt.Sprintf("*_%s", filepath.Base(h.Entity)))
				if err != nil {
					logger.Errorf("failed to create temporary file: %s", err)
					continue
				}

				c, err := NewClient(ctx, h.Entity)
				if err != nil {
					logger.Errorf("failed to create new remote client: %s", err)

					deleteLocalFile(ctx, tmpFile.Name())

					continue
				}

				err = c.DownloadFile(ctx, tmpFile.Name())
				if err != nil {
					logger.Errorf("failed to download file to temporary folder: %s", err)

					err = c.DownloadFileFallback(ctx, tmpFile.Name())
					if err != nil {
						logger.Errorf("failed to download remote file using fallback option: %s", err)
					}

					deleteLocalFile(ctx, tmpFile.Name())

					continue
				}

				h.LocalFile = tmpFile.Name()
				h.LocalFileNeedsCleanup = true

				filtered = append(filtered, h)
			}

			return next(ctx, filtered)
		}
	}
}

// WithCleanup initializes and returns a heartbeat handle option, which
// deletes a local temporary file if downloaded from a remote file.
func WithCleanup() heartbeat.HandleOption {
	return func(next heartbeat.Handle) heartbeat.Handle {
		return func(ctx context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			logger := log.Extract(ctx)
			logger.Debugln("execute remote cleanup")

			for _, h := range hh {
				if h.LocalFileNeedsCleanup {
					logger.Debugf("deleting temporary file: %s", h.LocalFile)

					deleteLocalFile(ctx, h.LocalFile)
				}
			}

			return next(ctx, hh)
		}
	}
}

func deleteLocalFile(ctx context.Context, fp string) {
	logger := log.Extract(ctx)

	err := os.Remove(fp)
	if err != nil {
		logger.Warnf("unable to delete tmp file: %s", err)
	}
}

// NewClient initializes a new remote client.
func NewClient(ctx context.Context, address string) (Client, error) {
	parsedURL, err := url.Parse(address)
	if err != nil {
		return Client{}, fmt.Errorf("failed to parse remote file url: %s", err)
	}

	host := parsedURL.Host
	pass, _ := parsedURL.User.Password()

	var port int

	if parsedURL.Port() != "" {
		// we're safe to ignore error here since `url.Parse` checks if port is valid integer
		port, _ = strconv.Atoi(parsedURL.Port())
		host = strings.Split(host, ":")[0]
	}

	derivedHost := ssh_config.Get(host, "HostName")
	if derivedHost == "" {
		derivedHost = host
	}

	logger := log.Extract(ctx)

	if port == 0 {
		port, err = strconv.Atoi(ssh_config.Get(host, "Port"))
		logger.Warnf("failed to parse port from host: %s", err)
	}

	if port == 0 {
		port, err = strconv.Atoi(ssh_config.Get(derivedHost, "Port"))
		logger.Warnf("failed to parse port from derived host: %s", err)
	}

	if port == 0 {
		port = defaultPort
	}

	return Client{
		User:         parsedURL.User.Username(),
		Pass:         pass,
		HostKeyAlias: hostKeyAlias(ctx, host, derivedHost),
		OriginalHost: host,
		Host:         derivedHost,
		Port:         port,
		Path:         parsedURL.Path,
	}, nil
}

// DownloadFile downloads a remote file and copy to a local file.
func (c Client) DownloadFile(ctx context.Context, localFile string) error {
	conn, sc, err := c.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to sftp host: %s", err)
	}

	logger := log.Extract(ctx)

	defer func() {
		if err := conn.Close(); err != nil {
			logger.Debugf("failed to close connection to ssh server: %s", err)
		}

		if err := sc.Close(); err != nil {
			logger.Debugf("failed to close connection to ftp server: %s", err)
		}
	}()

	srcFile, err := sc.OpenFile(c.Path, os.O_RDONLY)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %s", err)
	}

	defer func() {
		if err := srcFile.Close(); err != nil {
			logger.Debugf("failed to close remote ftp file: %s", err)
		}
	}()

	dstFile, err := os.Create(localFile) // nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to open local file: %s", err)
	}

	defer func() {
		if err := dstFile.Close(); err != nil {
			logger.Warnf("failed to close local file: %s", err)
		}
	}()

	_, err = io.CopyN(dstFile, srcFile, maxFileSize)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to download remote file: %s", err)
	}

	return nil
}

// DownloadFileFallback downloads a remote file and copy to a local file using machine's ssh.
func (c Client) DownloadFileFallback(ctx context.Context, localFile string) error {
	logger := log.Extract(ctx)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeoutSecs*time.Second)
	defer cancel()

	logger.Debugln("downloading remote file using fallback option")

	cmd := exec.CommandContext(ctx, "scp", "-B", fmt.Sprintf("%s:%s", c.OriginalHost, c.Path), localFile) // nolint:gosec

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp failed: %s. %s", &stderr, err)
	}

	return nil
}

// Connect connects to sftp host.
func (c Client) Connect(ctx context.Context) (*ssh.Client, *sftp.Client, error) {
	// Initialize client configuration
	sshClient, err := c.sshClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Create new SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start sftp subsystem: %s", err)
	}

	return sshClient, sftpClient, nil
}

// knownHostKeys gets all host keys from local known hosts for given hosts.
func (c Client) knownHostKeys(ctx context.Context) []ssh.PublicKey {
	logger := log.Extract(ctx)
	hostKeys := []ssh.PublicKey{}

	filenames := c.knownHostsFiles()

	for _, filename := range filenames {
		if err := func(fn string) error {
			file, err := file.OpenNoLock(fn) // nolint:gosec
			if err != nil {
				return fmt.Errorf("failed to open known_hosts file: %s", err)
			}

			defer func() {
				if err := file.Close(); err != nil {
					logger.Debugf("failed to close file '%s': %s", file.Name(), err)
				}
			}()

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				fields := strings.Split(scanner.Text(), " ")
				if len(fields) < 3 {
					continue
				}

				hostnames := strings.Split(fields[0], ",")

				if contains(hostnames, c.HostKeyAlias, c.OriginalHost, c.Host) {
					hostKey, _, _, _, err := ssh.ParseAuthorizedKey(scanner.Bytes())
					if err != nil {
						logger.Warnf("failed to parse %q: %s", fields[2], err)
					} else {
						hostKeys = append(hostKeys, hostKey)
					}
				}
			}

			return nil
		}(filename); err != nil {
			logger.Debugln(err.Error())
		}
	}

	return hostKeys
}

func (c Client) strictHostKeyChecking() string {
	strict := ssh_config.Get(c.OriginalHost, "StrictHostKeyChecking")

	if strict == "" && c.OriginalHost != c.Host {
		strict = ssh_config.Get(c.Host, "StrictHostKeyChecking")
	}

	if strict == "" {
		strict = ssh_config.Default("StrictHostKeyChecking")
	}

	if strict == "accept-new" || strict == "off" {
		strict = "no"
	}

	return strict
}

// knownHostsFiles returns paths to the known hosts files.
func (c Client) knownHostsFiles() []string {
	files := ssh_config.GetAll(c.OriginalHost, "UserKnownHostsFile")

	for _, f := range files {
		f, err := homedir.Expand(f)
		if err != nil {
			continue
		}

		files = append(files, f)
	}

	return files
}

// identityFile returns the path to a secret key file, or the first existing default.
func (c Client) identityFile() string {
	keyFiles := ssh_config.GetAll(c.OriginalHost, "IdentityFile")
	for _, key := range keyFiles {
		keyFile, err := homedir.Expand(key)
		if err != nil {
			continue
		}

		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			continue
		}

		return keyFile
	}

	if c.OriginalHost != c.Host {
		keyFiles := ssh_config.GetAll(c.Host, "IdentityFile")
		for _, key := range keyFiles {
			keyFile, err := homedir.Expand(key)
			if err != nil {
				continue
			}

			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				continue
			}

			return keyFile
		}
	}

	return ""
}

func (c Client) signerForIdentity() (ssh.Signer, error) {
	identityFile := c.identityFile()
	if identityFile == "" {
		return nil, nil
	}

	key, err := os.ReadFile(identityFile) // nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to read private key %s: %v", identityFile, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key %s: %v", identityFile, err)
	}

	return signer, nil
}

func (c Client) warnIfUsingRevokedHostKeys(ctx context.Context) {
	logger := log.Extract(ctx)

	revokedKeysFile := ssh_config.Get(c.OriginalHost, "RevokedHostKeys")
	if revokedKeysFile != "" {
		logger.Warnln("using ssh config RevokedHostKeys is not supported")
		return
	}

	if c.OriginalHost != c.Host {
		revokedKeysFile = ssh_config.Get(c.Host, "RevokedHostKeys")
		if revokedKeysFile != "" {
			logger.Warnln("using ssh config RevokedHostKeys is not supported")
		}
	}
}

func (c Client) sshClient(ctx context.Context) (*ssh.Client, error) {
	logger := log.Extract(ctx)

	var auths []ssh.AuthMethod

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	signer, err := c.signerForIdentity()
	if err != nil {
		logger.Warnf("%s", err)
	}

	if signer != nil {
		auths = append(auths, ssh.PublicKeys(signer))
	}

	// Try to use $SSH_AUTH_SOCK which contains the path of the unix file socket that the sshd agent uses
	// for communication with other processes
	if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
	}

	// Use password authentication if provided
	if c.Pass != "" {
		auths = append(auths, ssh.Password(c.Pass))
	}

	config := ssh.ClientConfig{
		User:    c.user(),
		Auth:    auths,
		Timeout: defaultTimeoutSecs * time.Second,
	}

	strict := c.strictHostKeyChecking()
	logger.Debugf("StrictHostKeyChecking for %s set to %s", c.OriginalHost, strict)

	if strict == "no" {
		logger.Debugf("host key checking disabled for %s", c.OriginalHost)

		config.HostKeyCallback = ssh.InsecureIgnoreHostKey() // nolint:gosec

		// Connect to server
		client, err := dial(addr, &config)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to '%s': %s", addr, err)
		}

		return client, nil
	}

	knownHostKeys := c.knownHostKeys(ctx)
	if len(knownHostKeys) == 0 && strict == "yes" {
		return nil, fmt.Errorf("known host key not found for %s, will not connect", c.OriginalHost)
	}

	if len(knownHostKeys) == 0 {
		logger.Debugf("no known host key found for %s, will connect anyway", c.OriginalHost)

		config.HostKeyCallback = ssh.InsecureIgnoreHostKey() // nolint:gosec

		// Connect to server
		client, err := dial(addr, &config)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to '%s': %s", addr, err)
		}

		return client, nil
	}

	logger.Debugf("found %d known host ssh keys for %s", len(knownHostKeys), c.OriginalHost)

	c.warnIfUsingRevokedHostKeys(ctx)

	for _, hostKey := range knownHostKeys {
		config.HostKeyCallback = ssh.FixedHostKey(hostKey)

		// Connect to server
		client, err := dial(addr, &config)
		if err != nil {
			logger.Warnf("failed to connect to '%s': %s", addr, err)

			continue
		}

		return client, nil
	}

	return nil, fmt.Errorf("failed to connect to %s", addr)
}

func (c Client) user() string {
	if c.User != "" {
		return c.User
	}

	if c.OriginalHost != "" {
		user := ssh_config.Get(c.OriginalHost, "User")
		if user != "" {
			return user
		}
	}

	if c.Host != c.OriginalHost {
		user := ssh_config.Get(c.Host, "User")
		if user != "" {
			return user
		}
	}

	return ""
}

func hostKeyAlias(ctx context.Context, hostOriginal, hostDerived string) string {
	alias := ssh_config.Get(hostOriginal, "HostKeyAlias")
	if alias == "" && hostOriginal != hostDerived {
		alias = ssh_config.Get(hostDerived, "HostKeyAlias")
	}

	if alias == "" {
		return ""
	}

	logger := log.Extract(ctx)

	alias, err := homedir.Expand(alias)
	if err != nil {
		logger.Debugf("Unable to expand home directory for HostKeyAlias %q: %s", alias, err)
	}

	return alias
}

// contains returns true if any case-insensitive arg is found in the list of values.
func contains(values []string, args ...string) bool {
	for _, val := range values {
		if val == "" {
			continue
		}

		val = strings.ToLower(val)

		for _, arg := range args {
			if val == strings.ToLower(arg) {
				return true
			}
		}
	}

	return false
}

func dial(addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial to '%s': %s", addr, err)
	}

	return conn, nil
}
