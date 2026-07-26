package sites

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const (
	kejilionWordPressArchiveURL = "https://raw.githubusercontent.com/kejilion/Website_source_code/refs/heads/main/wp-latest.zip"
	// This is the exact wp-latest.zip consumed by kejilion.sh when KPanel
	// v0.15 was built. Refuse silently changed remote code.
	kejilionWordPressArchiveSHA256 = "380e03fe346258e980dcfccdf6ee3db84400568b5a21d534e473c63e22c837e4"
	maxWordPressArchiveBytes       = 64 << 20
	maxWordPressExtractedBytes     = 256 << 20
	maxWordPressArchiveEntries     = 7000
	maxWordPressFileBytes          = 32 << 20
)

type WordPressDatabaseCredentials struct {
	Name     string
	User     string
	Password string
}

// WordPressRuntime exposes only the fixed database and certificate operations
// required by kejilion.sh's WordPress business flow.
type WordPressRuntime interface {
	WordPressReady(context.Context) error
	PrepareWordPressDatabase(context.Context, string) (WordPressDatabaseCredentials, error)
	RollbackWordPressDatabase(context.Context, string) error
	PrepareWordPressCertificate(context.Context, string) error
	RollbackWordPressCertificate(context.Context, string) error
}

type WordPressArchiveLoader func(context.Context) ([]byte, error)

func (m *Manager) WordPressWritable(ctx context.Context) error {
	if err := m.Writable(ctx); err != nil {
		return err
	}
	if m.wordPressRuntime == nil || m.archiveLoader == nil {
		return fmt.Errorf("%w: WordPress runtime is unavailable", ErrUnavailable)
	}
	if err := m.wordPressRuntime.WordPressReady(ctx); err != nil {
		return fmt.Errorf("%w: WordPress prerequisites are unavailable: %v", ErrUnavailable, err)
	}
	return nil
}

func normalizeWordPressInput(input SiteInput) (managedSpec, error) {
	primary, err := normalizeFQDN(input.PrimaryDomain)
	if err != nil {
		return managedSpec{}, err
	}
	if input.Enabled != nil && !*input.Enabled {
		return managedSpec{}, fmt.Errorf("%w: disabling WordPress is not supported", ErrUnprocessable)
	}
	if input.ExpectedResourceVersion != "" {
		return managedSpec{}, fmt.Errorf("%w: expectedResourceVersion is not valid for create", ErrInvalidInput)
	}
	if len(input.Aliases) != 0 {
		return managedSpec{}, fmt.Errorf("%w: the kejilion.sh WordPress flow accepts one primary domain", ErrUnprocessable)
	}
	if strings.TrimSpace(input.Upstream) != "" || len(input.Upstreams) != 0 ||
		strings.TrimSpace(input.RedirectTarget) != "" || input.RedirectCode != 0 ||
		strings.TrimSpace(input.PHPVersion) != "" {
		return managedSpec{}, fmt.Errorf("%w: WordPress cannot define generic site settings", ErrUnprocessable)
	}
	return managedSpec{
		Primary: primary, Kind: contract.SiteWordPress, PHPVersion: "latest",
	}, nil
}

type wordPressTransaction struct {
	spec              managedSpec
	configPath        string
	bootstrapConfig   []byte
	finalConfig       []byte
	bootstrapBackup   string
	configState       string
	documentRoot      string
	documentPublished bool
	documentHash      string
	databasePrepared  bool
	certificateReady  bool
}

func (m *Manager) installWordPress(ctx context.Context, input SiteInput) (contract.SiteSummary, error) {
	spec, err := normalizeWordPressInput(input)
	if err != nil {
		return contract.SiteSummary{}, err
	}

	siteWriteMutex.Lock()
	defer siteWriteMutex.Unlock()
	if err := m.WordPressWritable(ctx); err != nil {
		return contract.SiteSummary{}, err
	}
	if err := m.checkCollisions(spec, ""); err != nil {
		return contract.SiteSummary{}, err
	}

	archive, err := m.archiveLoader(ctx)
	if err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: load the pinned kejilion.sh WordPress package: %v", ErrUnavailable, err)
	}
	stagePath, err := stageWordPressArchive(filepath.Join(m.webRoot, "html"), spec.Primary, archive)
	if err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: stage WordPress files: %v", ErrUnprocessable, err)
	}
	defer func() {
		if stagePath != "" {
			_ = os.RemoveAll(stagePath)
		}
	}()

	databaseName := wordPressDatabaseName(spec.Primary)
	credentials, err := m.wordPressRuntime.PrepareWordPressDatabase(ctx, databaseName)
	if err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: create the kejilion.sh-compatible database: %v", ErrUnavailable, err)
	}
	tx := wordPressTransaction{
		spec:             spec,
		configPath:       filepath.Join(m.webRoot, "conf.d", spec.Primary+".conf"),
		bootstrapConfig:  renderWordPressBootstrapConfig(spec.Primary),
		finalConfig:      renderManagedConfig(spec),
		documentRoot:     filepath.Join(m.webRoot, "html", spec.Primary),
		databasePrepared: true,
	}
	fail := func(cause error) (contract.SiteSummary, error) {
		if rollbackErr := m.rollbackWordPress(ctx, &tx); rollbackErr != nil {
			return contract.SiteSummary{}, fmt.Errorf("%w: %v; rollback failed: %v", ErrNeedsAttention, cause, rollbackErr)
		}
		return contract.SiteSummary{}, cause
	}

	if credentials.Name != databaseName {
		return fail(fmt.Errorf("%w: database runtime returned an unexpected identity", ErrNeedsAttention))
	}
	if err := configureWordPress(stagePath, spec.Primary, credentials); err != nil {
		return fail(fmt.Errorf("%w: configure WordPress: %v", ErrUnavailable, err))
	}
	tx.documentHash, err = wordPressDirectoryFingerprint(stagePath)
	if err != nil {
		return fail(fmt.Errorf("%w: fingerprint staged WordPress files: %v", ErrUnavailable, err))
	}

	bootstrapTemp, err := writeTemporaryFile(
		filepath.Dir(tx.configPath), "."+spec.Primary+".kp-wp-bootstrap-*.tmp",
		tx.bootstrapConfig, 0o640,
	)
	if err != nil {
		return fail(fmt.Errorf("%w: stage ACME bootstrap configuration: %v", ErrUnavailable, err))
	}
	defer func() {
		if bootstrapTemp != "" {
			_ = os.Remove(bootstrapTemp)
		}
	}()
	if err := atomicNoReplace(bootstrapTemp, tx.configPath); err != nil {
		if pathExists(tx.configPath) {
			return fail(fmt.Errorf("%w: Nginx configuration already exists", ErrConflict))
		}
		return fail(fmt.Errorf("%w: publish ACME bootstrap configuration: %v", ErrUnavailable, err))
	}
	bootstrapTemp = ""
	tx.configState = "bootstrap"
	if err := syncDirectory(filepath.Dir(tx.configPath)); err != nil {
		return fail(fmt.Errorf("%w: sync ACME bootstrap configuration: %v", ErrUnavailable, err))
	}
	if err := m.nginx.NginxTest(ctx); err != nil {
		return fail(fmt.Errorf("%w: ACME bootstrap failed nginx -t: %v", ErrUnprocessable, err))
	}
	if err := m.nginx.NginxReload(ctx); err != nil {
		return fail(fmt.Errorf("%w: activate ACME bootstrap: %v", ErrUnavailable, err))
	}

	if err := m.wordPressRuntime.PrepareWordPressCertificate(ctx, spec.Primary); err != nil {
		return fail(fmt.Errorf("%w: issue the WordPress certificate: %v", ErrUnavailable, err))
	}
	tx.certificateReady = true

	if err := atomicNoReplace(stagePath, tx.documentRoot); err != nil {
		return fail(fmt.Errorf("%w: publish WordPress document root: %v", ErrUnavailable, err))
	}
	stagePath = ""
	tx.documentPublished = true
	if err := syncDirectory(filepath.Dir(tx.documentRoot)); err != nil {
		return fail(fmt.Errorf("%w: sync WordPress document root: %v", ErrUnavailable, err))
	}

	finalTemp, err := writeTemporaryFile(
		filepath.Dir(tx.configPath), "."+spec.Primary+".kp-wp-final-*.tmp",
		tx.finalConfig, 0o640,
	)
	if err != nil {
		return fail(fmt.Errorf("%w: stage final WordPress configuration: %v", ErrUnavailable, err))
	}
	tx.bootstrapBackup = finalTemp
	if err := atomicExchange(finalTemp, tx.configPath); err != nil {
		return fail(fmt.Errorf("%w: activate final WordPress configuration: %v", ErrUnavailable, err))
	}
	tx.configState = "final"
	if err := syncDirectory(filepath.Dir(tx.configPath)); err != nil {
		return fail(fmt.Errorf("%w: sync final WordPress configuration: %v", ErrUnavailable, err))
	}
	if err := m.nginx.NginxTest(ctx); err != nil {
		return fail(fmt.Errorf("%w: final WordPress configuration failed nginx -t: %v", ErrUnprocessable, err))
	}
	if err := m.nginx.NginxReload(ctx); err != nil {
		return fail(fmt.Errorf("%w: activate final WordPress website: %v", ErrUnavailable, err))
	}
	if err := os.Remove(tx.bootstrapBackup); err != nil {
		return contract.SiteSummary{}, fmt.Errorf(
			"%w: WordPress is active but bootstrap backup cleanup failed at %s: %v",
			ErrNeedsAttention, tx.bootstrapBackup, err,
		)
	}
	tx.bootstrapBackup = ""
	tx.configState = "committed"
	if err := syncDirectory(filepath.Dir(tx.configPath)); err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: WordPress is active but configuration cleanup was not synced: %v", ErrNeedsAttention, err)
	}
	return m.discoverManaged(spec.Primary)
}

func (m *Manager) rollbackWordPress(ctx context.Context, tx *wordPressTransaction) error {
	var failures []string
	backupExpected := tx.bootstrapConfig
	if tx.configState == "final" {
		if !pathMatchesBytes(tx.configPath, tx.finalConfig) ||
			!pathMatchesBytes(tx.bootstrapBackup, tx.bootstrapConfig) {
			failures = append(failures, "Nginx transaction changed externally")
		} else if err := atomicExchange(tx.configPath, tx.bootstrapBackup); err != nil {
			failures = append(failures, "restore ACME bootstrap: "+err.Error())
		} else {
			tx.configState = "bootstrap"
			backupExpected = tx.finalConfig
		}
	}
	if tx.configState == "bootstrap" {
		if !pathMatchesBytes(tx.configPath, tx.bootstrapConfig) {
			failures = append(failures, "ACME bootstrap changed externally")
		} else if err := os.Remove(tx.configPath); err != nil {
			failures = append(failures, "remove ACME bootstrap: "+err.Error())
		} else {
			tx.configState = ""
			if err := syncDirectory(filepath.Dir(tx.configPath)); err != nil {
				failures = append(failures, "sync Nginx rollback: "+err.Error())
			}
		}
	}
	if tx.bootstrapBackup != "" {
		if pathMatchesBytes(tx.bootstrapBackup, backupExpected) {
			if err := os.Remove(tx.bootstrapBackup); err != nil {
				failures = append(failures, "remove bootstrap backup: "+err.Error())
			}
		} else if pathExists(tx.bootstrapBackup) {
			failures = append(failures, "bootstrap backup changed externally")
		}
	}
	if tx.configState == "" {
		if err := m.nginx.NginxTest(ctx); err != nil {
			failures = append(failures, "validate previous Nginx configuration: "+err.Error())
		} else if err := m.nginx.NginxReload(ctx); err != nil {
			failures = append(failures, "reload previous Nginx configuration: "+err.Error())
		}
	}
	if tx.documentPublished {
		currentHash, err := wordPressDirectoryFingerprint(tx.documentRoot)
		if err != nil || currentHash != tx.documentHash {
			failures = append(failures, "WordPress document root changed externally")
		} else if err := os.RemoveAll(tx.documentRoot); err != nil {
			failures = append(failures, "remove created WordPress document root: "+err.Error())
		} else if err := syncDirectory(filepath.Dir(tx.documentRoot)); err != nil {
			failures = append(failures, "sync WordPress document rollback: "+err.Error())
		}
	}
	if tx.certificateReady {
		if err := m.wordPressRuntime.RollbackWordPressCertificate(ctx, tx.spec.Primary); err != nil {
			failures = append(failures, "rollback copied certificate: "+err.Error())
		}
	}
	if tx.databasePrepared {
		if err := m.wordPressRuntime.RollbackWordPressDatabase(ctx, wordPressDatabaseName(tx.spec.Primary)); err != nil {
			failures = append(failures, "rollback created database: "+err.Error())
		}
	}
	if len(failures) != 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func downloadKejilionWordPressArchive(ctx context.Context) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, kejilionWordPressArchiveURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/zip, application/octet-stream")
	client := &http.Client{Timeout: 5 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxWordPressArchiveBytes {
		return nil, errors.New("WordPress archive exceeds the safety limit")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxWordPressArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxWordPressArchiveBytes {
		return nil, errors.New("WordPress archive exceeds the safety limit")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != kejilionWordPressArchiveSHA256 {
		return nil, errors.New("WordPress archive checksum changed; update KPanel before installing")
	}
	return data, nil
}

func stageWordPressArchive(parent, primary string, archive []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return "", err
	}
	if len(reader.File) == 0 || len(reader.File) > maxWordPressArchiveEntries {
		return "", errors.New("WordPress archive entry count is outside the safety limit")
	}
	stage, err := os.MkdirTemp(parent, "."+primary+".kp-wordpress-*")
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(stage)
		}
	}()
	var total uint64
	for _, entry := range reader.File {
		if entry.FileInfo().Mode()&os.ModeSymlink != 0 {
			return "", errors.New("WordPress archive contains a symbolic link")
		}
		name := filepath.ToSlash(entry.Name)
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
		if clean == "." || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "../") ||
			(clean != "wordpress" && !strings.HasPrefix(clean, "wordpress/")) {
			return "", fmt.Errorf("WordPress archive contains an unsafe path %q", name)
		}
		target := filepath.Join(stage, filepath.FromSlash(clean))
		if !pathWithin(target, stage) {
			return "", fmt.Errorf("WordPress archive path escaped the staging root")
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if entry.UncompressedSize64 > maxWordPressFileBytes ||
			total+entry.UncompressedSize64 > maxWordPressExtractedBytes {
			return "", errors.New("WordPress archive expands beyond the safety limit")
		}
		total += entry.UncompressedSize64
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		source, err := entry.Open()
		if err != nil {
			return "", err
		}
		destination, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			source.Close()
			return "", err
		}
		_, copyErr := io.Copy(destination, io.LimitReader(source, maxWordPressFileBytes+1))
		closeErr := destination.Close()
		sourceErr := source.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		if sourceErr != nil {
			return "", sourceErr
		}
	}
	sample := filepath.Join(stage, "wordpress", "wp-config-sample.php")
	info, err := os.Lstat(sample)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("WordPress package is missing wp-config-sample.php")
	}
	ok = true
	return stage, nil
}

func configureWordPress(stage, domain string, credentials WordPressDatabaseCredentials) error {
	samplePath := filepath.Join(stage, "wordpress", "wp-config-sample.php")
	sample, err := os.ReadFile(samplePath)
	if err != nil {
		return err
	}
	config := string(sample)
	replacements := map[string]string{
		"database_name_here": phpSingleQuoted(credentials.Name),
		"username_here":      phpSingleQuoted(credentials.User),
		"password_here":      phpSingleQuoted(credentials.Password),
		"localhost":          "mysql",
	}
	for old, replacement := range replacements {
		if !strings.Contains(config, old) {
			return fmt.Errorf("WordPress config placeholder %q is missing", old)
		}
		config = strings.Replace(config, old, replacement, 1)
	}
	for index := 0; index < 8; index++ {
		const placeholder = "put your unique phrase here"
		if !strings.Contains(config, placeholder) {
			return errors.New("WordPress salt placeholders are incomplete")
		}
		salt := make([]byte, 48)
		if _, err := rand.Read(salt); err != nil {
			return err
		}
		config = strings.Replace(config, placeholder, base64.RawURLEncoding.EncodeToString(salt), 1)
	}
	const marker = "/* That's all, stop editing!"
	if !strings.Contains(config, marker) {
		return errors.New("WordPress custom configuration marker is missing")
	}
	custom := "define('FS_METHOD', 'direct');\n" +
		"define('WP_REDIS_HOST', 'redis');\n" +
		"define('WP_REDIS_PORT', '6379');\n" +
		"define('WP_REDIS_MAXTTL', 86400);\n" +
		"define('WP_CACHE_KEY_SALT', '" + phpSingleQuoted(domain) + "_');\n" +
		"define('WP_HOME', 'https://" + phpSingleQuoted(domain) + "');\n" +
		"define('WP_SITEURL', 'https://" + phpSingleQuoted(domain) + "');\n\n"
	config = strings.Replace(config, marker, custom+marker, 1)
	path := filepath.Join(stage, "wordpress", "wp-config.php")
	temp, err := writeTemporaryFile(filepath.Dir(path), ".wp-config-*.tmp", []byte(config), 0o644)
	if err != nil {
		return err
	}
	if err := atomicNoReplace(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func phpSingleQuoted(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `'`, `\'`)
}

func wordPressDatabaseName(domain string) string {
	var builder strings.Builder
	for _, character := range domain {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') {
			builder.WriteRune(character)
		} else {
			builder.WriteByte('_')
		}
	}
	return builder.String()
}

func wordPressDirectoryFingerprint(root string) (string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("WordPress tree contains a symbolic link")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative != "." {
			paths = append(paths, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, relative := range paths {
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		hash.Write([]byte(relative))
		if info.IsDir() {
			hash.Write([]byte{0})
			continue
		}
		if !info.Mode().IsRegular() {
			return "", errors.New("WordPress tree contains a non-regular file")
		}
		hash.Write([]byte{1})
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(hash, file); err != nil {
			file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func pathMatchesBytes(path string, expected []byte) bool {
	info, err := os.Lstat(path)
	return err == nil && fileMatches(path, info, expected)
}

func pathWithin(candidate, root string) bool {
	candidate = filepath.Clean(candidate)
	root = filepath.Clean(root)
	return candidate == root || strings.HasPrefix(candidate, root+string(filepath.Separator))
}
