package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// GitProtocol handles Git smart HTTP protocol operations
type GitProtocol struct{}

// NewGitProtocol creates a new GitProtocol instance
func NewGitProtocol() *GitProtocol {
	return &GitProtocol{}
}

// ServiceType represents the type of Git service
type ServiceType string

const (
	// ServiceUploadPack is the service for git fetch/clone operations
	ServiceUploadPack ServiceType = "git-upload-pack"

	// ServiceReceivePack is the service for git push operations
	ServiceReceivePack ServiceType = "git-receive-pack"
)

// InfoRefsRequest represents a request for info/refs
type InfoRefsRequest struct {
	RepoPath string
	Service  ServiceType
}

// InfoRefsResponse represents the response for info/refs
type InfoRefsResponse struct {
	ContentType string
	Body        []byte
}

// PackRequest represents a request for upload-pack or receive-pack
type PackRequest struct {
	RepoPath string
	Service  ServiceType
	Body     io.Reader
}

// ContentTypeForService returns the content type for a service
func ContentTypeForService(service ServiceType) string {
	return fmt.Sprintf("application/x-%s-result", service)
}

// AdvertisementContentType returns the content type for service advertisement
func AdvertisementContentType(service ServiceType) string {
	return fmt.Sprintf("application/x-%s-advertisement", service)
}

// IsValidService checks if the service name is valid
func IsValidService(service string) bool {
	return service == string(ServiceUploadPack) || service == string(ServiceReceivePack)
}

// NormalizeServiceName ensures the service name has the git- prefix
func NormalizeServiceName(service string) ServiceType {
	if !strings.HasPrefix(service, "git-") {
		service = "git-" + service
	}
	return ServiceType(service)
}

// GetInfoRefs returns the info/refs response for smart HTTP protocol
func (p *GitProtocol) GetInfoRefs(ctx context.Context, req InfoRefsRequest) (*InfoRefsResponse, error) {
	var buf bytes.Buffer

	// Write pkt-line header for service advertisement
	header := fmt.Sprintf("# service=%s\n", req.Service)
	pktHeader := EncodePktLine(header)
	buf.WriteString(pktHeader)
	buf.WriteString("0000") // Flush packet

	// Get refs using git command
	// Remove "git-" prefix from service name (e.g., "git-receive-pack" -> "receive-pack")
	serviceName := strings.TrimPrefix(string(req.Service), "git-")
	cmd := exec.CommandContext(ctx, "git", serviceName, "--stateless-rpc", "--advertise-refs", req.RepoPath)
	cmd.Dir = req.RepoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to advertise refs: %w (stderr: %s, repoPath: %s)", err, stderr.String(), req.RepoPath)
	}

	buf.Write(stdout.Bytes())

	return &InfoRefsResponse{
		ContentType: AdvertisementContentType(req.Service),
		Body:        buf.Bytes(),
	}, nil
}

// HandleUploadPack handles git-upload-pack for fetch/clone operations
func (p *GitProtocol) HandleUploadPack(ctx context.Context, repoPath string, input io.Reader, output io.Writer) error {
	return p.runGitService(ctx, repoPath, ServiceUploadPack, input, output, true)
}

// HandleReceivePack handles git-receive-pack for push operations
func (p *GitProtocol) HandleReceivePack(ctx context.Context, repoPath string, input io.Reader, output io.Writer) error {
	err := p.runGitService(ctx, repoPath, ServiceReceivePack, input, output, true)
	if err != nil {
		return err
	}

	// Update server info after receiving push
	if err := p.updateServerInfo(ctx, repoPath); err != nil {
		// TODO: log error but do not fail the push
	}

	return nil
}

// HandleUploadPackSSH handles git-upload-pack for SSH transport (stateful)
func (p *GitProtocol) HandleUploadPackSSH(ctx context.Context, repoPath string, input io.Reader, output io.Writer) error {
	return p.runGitService(ctx, repoPath, ServiceUploadPack, input, output, false)
}

// HandleReceivePackSSH handles git-receive-pack for SSH transport (stateful)
func (p *GitProtocol) HandleReceivePackSSH(ctx context.Context, repoPath string, input io.Reader, output io.Writer) error {
	err := p.runGitService(ctx, repoPath, ServiceReceivePack, input, output, false)
	if err != nil {
		return err
	}

	// Update server info after receiving push
	if err := p.updateServerInfo(ctx, repoPath); err != nil {
		// TODO: log error but do not fail the push
	}

	return nil
}

// runGitService executes a git service command
func (p *GitProtocol) runGitService(ctx context.Context, repoPath string, service ServiceType, input io.Reader, output io.Writer, stateless bool) error {
	// Remove "git-" prefix from service name (e.g., "git-receive-pack" -> "receive-pack")
	serviceName := strings.TrimPrefix(string(service), "git-")

	args := []string{serviceName}
	if stateless {
		args = append(args, "--stateless-rpc")
	}
	args = append(args, repoPath)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	cmd.Stdin = input
	cmd.Stdout = output

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", service, err)
	}

	return nil
}

// updateServerInfo updates auxiliary info file (for dumb HTTP protocol)
func (p *GitProtocol) updateServerInfo(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "update-server-info")
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update server info: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// EncodePktLine encodes a string as a pkt-line
func EncodePktLine(data string) string {
	length := len(data) + 4
	return fmt.Sprintf("%04x%s", length, data)
}

// DecodePktLine decodes a pkt-line from a reader
func DecodePktLine(r io.Reader) (string, error) {
	// Read length prefix (4 hex characters)
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return "", err
	}

	// Parse length
	var length int
	_, err := fmt.Sscanf(string(lenBuf), "%04x", &length)
	if err != nil {
		return "", fmt.Errorf("invalid pkt-line length: %w", err)
	}

	// Flush packet
	if length == 0 {
		return "", nil
	}

	// Read data (length includes the 4-byte prefix)
	dataBuf := make([]byte, length-4)
	if _, err := io.ReadFull(r, dataBuf); err != nil {
		return "", err
	}

	return string(dataBuf), nil
}

// FlushPacket returns the flush packet
func FlushPacket() string {
	return "0000"
}

// DelimPacket returns the delimiter packet (used in protocol v2)
func DelimPacket() string {
	return "0001"
}

// ResponseEndPacket returns the response end packet (used in protocol v2)
func ResponseEndPacket() string {
	return "0002"
}

// PktLineWriter is a writer that encodes data as pkt-lines
type PktLineWriter struct {
	w io.Writer
}

// NewPktLineWriter creates a new PktLineWriter
func NewPktLineWriter(w io.Writer) *PktLineWriter {
	return &PktLineWriter{w: w}
}

// WriteLine writes a line as a pkt-line
func (w *PktLineWriter) WriteLine(line string) error {
	_, err := w.w.Write([]byte(EncodePktLine(line)))
	return err
}

// WriteFlush writes a flush packet
func (w *PktLineWriter) WriteFlush() error {
	_, err := w.w.Write([]byte(FlushPacket()))
	return err
}

// WriteDelim writes a delimiter packet
func (w *PktLineWriter) WriteDelim() error {
	_, err := w.w.Write([]byte(DelimPacket()))
	return err
}

// WriteData writes binary data as a pkt-line
func (w *PktLineWriter) WriteData(data []byte) error {
	length := len(data) + 4
	header := fmt.Sprintf("%04x", length)
	if _, err := w.w.Write([]byte(header)); err != nil {
		return err
	}
	_, err := w.w.Write(data)
	return err
}

// PktLineReader is a reader that decodes pkt-lines
type PktLineReader struct {
	r io.Reader
}

// NewPktLineReader creates a new PktLineReader
func NewPktLineReader(r io.Reader) *PktLineReader {
	return &PktLineReader{r: r}
}

// ReadLine reads a pkt-line
func (r *PktLineReader) ReadLine() (string, error) {
	return DecodePktLine(r.r)
}

// ReadAll reads all pkt-lines until a flush packet
func (r *PktLineReader) ReadAll() ([]string, error) {
	var lines []string
	for {
		line, err := r.ReadLine()
		if err != nil {
			return lines, err
		}
		if line == "" {
			// Flush packet
			break
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// Capabilities represents Git protocol capabilities
type Capabilities struct {
	values map[string]string
}

// NewCapabilities creates a new Capabilities instance
func NewCapabilities() *Capabilities {
	return &Capabilities{
		values: make(map[string]string),
	}
}

// Add adds a capability
func (c *Capabilities) Add(name, value string) {
	c.values[name] = value
}

// Get gets a capability value
func (c *Capabilities) Get(name string) (string, bool) {
	v, ok := c.values[name]
	return v, ok
}

// Has checks if a capability exists
func (c *Capabilities) Has(name string) bool {
	_, ok := c.values[name]
	return ok
}

// String returns the capabilities as a string
func (c *Capabilities) String() string {
	var parts []string
	for k, v := range c.values {
		if v == "" {
			parts = append(parts, k)
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return strings.Join(parts, " ")
}

// ParseCapabilities parses capabilities from a string
func ParseCapabilities(s string) *Capabilities {
	caps := NewCapabilities()
	parts := strings.Fields(s)
	for _, part := range parts {
		if idx := strings.Index(part, "="); idx != -1 {
			caps.Add(part[:idx], part[idx+1:])
		} else {
			caps.Add(part, "")
		}
	}
	return caps
}

// DefaultCapabilities returns default server capabilities
func DefaultCapabilities() *Capabilities {
	caps := NewCapabilities()
	caps.Add("multi_ack", "")
	caps.Add("thin-pack", "")
	caps.Add("side-band", "")
	caps.Add("side-band-64k", "")
	caps.Add("ofs-delta", "")
	caps.Add("shallow", "")
	caps.Add("no-progress", "")
	caps.Add("include-tag", "")
	caps.Add("multi_ack_detailed", "")
	caps.Add("no-done", "")
	caps.Add("symref", "HEAD:refs/heads/main")
	caps.Add("agent", "git-server/1.0")
	return caps
}

// SideBandChannel represents side-band channel types
type SideBandChannel byte

const (
	// SideBandData is the main data channel
	SideBandData SideBandChannel = 1

	// SideBandProgress is the progress message channel
	SideBandProgress SideBandChannel = 2

	// SideBandError is the error message channel
	SideBandError SideBandChannel = 3
)

// WriteSideBand writes data to a side-band channel
func WriteSideBand(w io.Writer, channel SideBandChannel, data []byte) error {
	// Format: length (4 hex) + channel byte + data
	length := len(data) + 5 // 4 for length + 1 for channel
	header := fmt.Sprintf("%04x", length)
	if _, err := w.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := w.Write([]byte{byte(channel)}); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// WriteSideBandProgress writes a progress message
func WriteSideBandProgress(w io.Writer, message string) error {
	return WriteSideBand(w, SideBandProgress, []byte(message))
}

// WriteSideBandError writes an error message
func WriteSideBandError(w io.Writer, message string) error {
	return WriteSideBand(w, SideBandError, []byte(message))
}
