package sites

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

type NginxController interface {
	NginxReady(context.Context) error
	NginxTest(context.Context) error
	NginxReload(context.Context) error
}

type Manager struct {
	webRoot          string
	discoverer       *Discoverer
	nginx            NginxController
	wordPressRuntime WordPressRuntime
	archiveLoader    WordPressArchiveLoader
	wordPressJobs    *wordPressJobRegistry
	recipeJobs       *recipeJobRegistry
	testHook         func(stage, path string)
}

var siteWriteMutex sync.Mutex

func NewManager(webRoot string, discoverer *Discoverer, nginx NginxController) *Manager {
	if discoverer == nil {
		discoverer = NewDiscoverer(webRoot)
	}
	manager := &Manager{
		webRoot: filepath.Clean(webRoot), discoverer: discoverer, nginx: nginx,
	}
	if runtime, ok := nginx.(WordPressRuntime); ok {
		manager.wordPressRuntime = runtime
	}
	manager.archiveLoader = downloadKejilionWordPressArchive
	manager.wordPressJobs = newWordPressJobRegistry("")
	manager.recipeJobs = newRecipeJobRegistry("")
	return manager
}

func (m *Manager) Writable(ctx context.Context) error {
	if !atomicSiteWritesSupported() {
		return fmt.Errorf("%w: atomic site transactions require Linux", ErrUnavailable)
	}
	if !filepath.IsAbs(m.webRoot) || m.webRoot == string(filepath.Separator) {
		return fmt.Errorf("%w: managed Web root must be a dedicated absolute directory", ErrUnavailable)
	}
	if m.nginx == nil {
		return fmt.Errorf("%w: Nginx controller is unavailable", ErrUnavailable)
	}
	for _, path := range []string{
		m.webRoot,
		filepath.Join(m.webRoot, "conf.d"),
		filepath.Join(m.webRoot, "html"),
	} {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: managed directory %s is unavailable or unsafe", ErrUnavailable, path)
		}
	}
	if err := m.nginx.NginxReady(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if err := m.nginx.NginxTest(ctx); err != nil {
		return fmt.Errorf("%w: existing Nginx configuration is invalid: %v", ErrUnavailable, err)
	}
	return nil
}

func (m *Manager) Create(ctx context.Context, input SiteInput) (contract.SiteSummary, error) {
	if input.Type == "wordpress" {
		return m.installWordPress(ctx, input)
	}
	spec, err := normalizeSiteInput(input)
	if err != nil {
		return contract.SiteSummary{}, err
	}
	if input.ExpectedResourceVersion != "" {
		return contract.SiteSummary{}, fmt.Errorf("%w: expectedResourceVersion is not valid for create", ErrInvalidInput)
	}

	siteWriteMutex.Lock()
	defer siteWriteMutex.Unlock()
	if err := m.Writable(ctx); err != nil {
		return contract.SiteSummary{}, err
	}
	if err := m.checkCollisions(spec, ""); err != nil {
		return contract.SiteSummary{}, err
	}

	config := renderManagedConfig(spec)
	configPath := filepath.Join(m.webRoot, "conf.d", spec.Primary+".conf")
	configTemp, err := writeTemporaryFile(filepath.Dir(configPath), "."+spec.Primary+".kp-*.tmp", config, 0o640)
	if err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: stage Nginx configuration: %v", ErrUnavailable, err)
	}
	defer os.Remove(configTemp)

	var staticPath, staticTemp string
	var staticInfo os.FileInfo
	if siteNeedsDocumentRoot(spec.Kind) {
		staticPath = filepath.Join(m.webRoot, "html", spec.Primary)
		indexName, indexBody := managedDefaultDocument(spec)
		staticTemp, err = stageStaticDirectory(
			filepath.Dir(staticPath),
			spec.Primary,
			indexName,
			indexBody,
		)
		if err != nil {
			return contract.SiteSummary{}, fmt.Errorf("%w: stage document root: %v", ErrUnavailable, err)
		}
		defer removeStagedStaticDirectory(staticTemp, indexName)
		if err := atomicNoReplace(staticTemp, staticPath); err != nil {
			if pathExists(staticPath) {
				return contract.SiteSummary{}, fmt.Errorf("%w: document root already exists", ErrConflict)
			}
			return contract.SiteSummary{}, fmt.Errorf("%w: publish document root: %v", ErrUnavailable, err)
		}
		staticTemp = ""
		staticInfo, _ = os.Lstat(staticPath)
		if err := syncDirectory(filepath.Dir(staticPath)); err != nil {
			if rollbackErr := m.removeCreatedStatic(staticPath, staticInfo, spec); rollbackErr != nil {
				return contract.SiteSummary{}, rollbackErr
			}
			return contract.SiteSummary{}, fmt.Errorf("%w: sync document root publication; candidate rolled back: %v", ErrUnavailable, err)
		}
	}

	m.callHook("before_config_publish", configPath)
	if err := atomicNoReplace(configTemp, configPath); err != nil {
		rollbackErr := m.removeCreatedStatic(staticPath, staticInfo, spec)
		if rollbackErr != nil {
			return contract.SiteSummary{}, rollbackErr
		}
		if pathExists(configPath) {
			return contract.SiteSummary{}, fmt.Errorf("%w: Nginx configuration already exists", ErrConflict)
		}
		return contract.SiteSummary{}, fmt.Errorf("%w: publish Nginx configuration: %v", ErrUnavailable, err)
	}
	configTemp = ""
	configInfo, _ := os.Lstat(configPath)
	if err := syncDirectory(filepath.Dir(configPath)); err != nil {
		if rollbackErr := m.rollbackCreate(configPath, configInfo, config, staticPath, staticInfo, spec); rollbackErr != nil {
			return contract.SiteSummary{}, rollbackErr
		}
		return contract.SiteSummary{}, fmt.Errorf("%w: sync configuration publication; candidate rolled back: %v", ErrUnavailable, err)
	}
	m.callHook("candidate_published", configPath)

	if !fileMatches(configPath, configInfo, config) {
		return contract.SiteSummary{}, fmt.Errorf("%w: candidate changed before validation", ErrNeedsAttention)
	}
	if err := m.nginx.NginxTest(ctx); err != nil {
		if rollbackErr := m.rollbackCreate(configPath, configInfo, config, staticPath, staticInfo, spec); rollbackErr != nil {
			return contract.SiteSummary{}, rollbackErr
		}
		return contract.SiteSummary{}, fmt.Errorf("%w: candidate failed nginx -t: %v", ErrUnprocessable, err)
	}
	if !fileMatches(configPath, configInfo, config) {
		return contract.SiteSummary{}, fmt.Errorf("%w: candidate changed during validation", ErrNeedsAttention)
	}
	if err := m.nginx.NginxReload(ctx); err != nil {
		if rollbackErr := m.rollbackCreate(configPath, configInfo, config, staticPath, staticInfo, spec); rollbackErr != nil {
			return contract.SiteSummary{}, rollbackErr
		}
		if recoveryErr := m.validateAndReloadPrevious(ctx); recoveryErr != nil {
			return contract.SiteSummary{}, recoveryErr
		}
		return contract.SiteSummary{}, fmt.Errorf("%w: Nginx reload failed and the candidate was rolled back: %v", ErrUnavailable, err)
	}
	if !fileMatches(configPath, configInfo, config) {
		return contract.SiteSummary{}, fmt.Errorf("%w: candidate changed while reloading", ErrNeedsAttention)
	}
	return m.discoverManaged(spec.Primary)
}

func (m *Manager) Update(ctx context.Context, id string, input SiteInput) (contract.SiteSummary, error) {
	spec, err := normalizeSiteInput(input)
	if err != nil {
		return contract.SiteSummary{}, err
	}
	if input.ExpectedResourceVersion == "" {
		return contract.SiteSummary{}, fmt.Errorf("%w: expectedResourceVersion is required", ErrInvalidInput)
	}

	siteWriteMutex.Lock()
	defer siteWriteMutex.Unlock()
	if err := m.Writable(ctx); err != nil {
		return contract.SiteSummary{}, err
	}
	current, err := m.findManagedByID(id)
	if err != nil {
		return contract.SiteSummary{}, err
	}
	if current.PrimaryDomain != spec.Primary || current.Kind != spec.Kind {
		return contract.SiteSummary{}, fmt.Errorf("%w: primary domain and site type are immutable", ErrForbidden)
	}
	if current.ResourceVersion != input.ExpectedResourceVersion {
		return contract.SiteSummary{}, fmt.Errorf("%w: resource version changed", ErrConflict)
	}
	if err := m.checkCollisions(spec, current.ID); err != nil {
		return contract.SiteSummary{}, err
	}
	if current.Origin == contract.OriginCLI {
		return m.updateCLIConfig(ctx, current, spec, input.ExpectedResourceVersion)
	}
	if siteNeedsDocumentRoot(spec.Kind) {
		rootInfo, rootErr := os.Lstat(filepath.Join(m.webRoot, "html", spec.Primary))
		if rootErr != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
			return contract.SiteSummary{}, fmt.Errorf("%w: document root is unavailable or unsafe", ErrForbidden)
		}
	}

	oldSpec, err := managedSpecFromSummary(current)
	if err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: current site is no longer canonical", ErrForbidden)
	}
	oldConfig := renderManagedConfig(oldSpec)
	newConfig := renderManagedConfig(spec)
	configPath := filepath.Join(m.webRoot, "conf.d", spec.Primary+".conf")
	oldInfo, err := os.Lstat(configPath)
	if err == nil && !fileMatches(configPath, oldInfo, oldConfig) &&
		(oldSpec.Kind == contract.SiteStatic || oldSpec.Kind == contract.SiteReverseProxy) {
		legacyConfig := renderManagedConfigV1(oldSpec)
		if fileMatches(configPath, oldInfo, legacyConfig) {
			oldConfig = legacyConfig
		}
	}
	if err != nil || !fileMatches(configPath, oldInfo, oldConfig) {
		return contract.SiteSummary{}, fmt.Errorf("%w: current configuration is no longer canonical", ErrForbidden)
	}
	if bytes.Equal(oldConfig, newConfig) {
		return current, nil
	}

	candidatePath, err := writeTemporaryFile(filepath.Dir(configPath), "."+spec.Primary+".kp-*.tmp", newConfig, 0o640)
	if err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: stage Nginx configuration: %v", ErrUnavailable, err)
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			_ = os.Remove(candidatePath)
		}
	}()

	latest, err := m.findManagedByID(id)
	if err != nil || latest.ResourceVersion != input.ExpectedResourceVersion ||
		!fileMatches(configPath, oldInfo, oldConfig) {
		return contract.SiteSummary{}, fmt.Errorf("%w: resource changed before commit", ErrConflict)
	}
	m.callHook("before_exchange", configPath)
	if err := atomicExchange(candidatePath, configPath); err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: atomically replace configuration: %v", ErrUnavailable, err)
	}
	candidateInfo, candidateInfoErr := os.Lstat(configPath)
	if err := syncDirectory(filepath.Dir(configPath)); err != nil {
		if candidateInfoErr != nil {
			keepBackup = true
			return contract.SiteSummary{}, fmt.Errorf("%w: inspect unsynced candidate; backup retained at %s: %v", ErrNeedsAttention, candidatePath, candidateInfoErr)
		}
		if restoreErr := restoreExchange(configPath, candidateInfo, newConfig, candidatePath); restoreErr != nil {
			keepBackup = true
			return contract.SiteSummary{}, restoreErr
		}
		return contract.SiteSummary{}, fmt.Errorf("%w: sync candidate publication; previous configuration restored: %v", ErrUnavailable, err)
	}
	if candidateInfoErr != nil {
		keepBackup = true
		return contract.SiteSummary{}, fmt.Errorf("%w: inspect published candidate; backup retained at %s: %v", ErrNeedsAttention, candidatePath, candidateInfoErr)
	}
	backupInfo, _ := os.Lstat(candidatePath)
	if !fileMatches(candidatePath, backupInfo, oldConfig) {
		if restoreErr := restoreExchange(configPath, candidateInfo, newConfig, candidatePath); restoreErr != nil {
			keepBackup = true
			return contract.SiteSummary{}, restoreErr
		}
		return contract.SiteSummary{}, fmt.Errorf("%w: external change won the update race", ErrConflict)
	}
	m.callHook("candidate_published", configPath)

	if !fileMatches(configPath, candidateInfo, newConfig) {
		keepBackup = true
		return contract.SiteSummary{}, fmt.Errorf("%w: candidate changed before validation; backup retained at %s", ErrNeedsAttention, candidatePath)
	}
	if err := m.nginx.NginxTest(ctx); err != nil {
		if restoreErr := restoreExchange(configPath, candidateInfo, newConfig, candidatePath); restoreErr != nil {
			keepBackup = true
			return contract.SiteSummary{}, restoreErr
		}
		return contract.SiteSummary{}, fmt.Errorf("%w: candidate failed nginx -t: %v", ErrUnprocessable, err)
	}
	if !fileMatches(configPath, candidateInfo, newConfig) {
		keepBackup = true
		return contract.SiteSummary{}, fmt.Errorf("%w: candidate changed during validation; backup retained at %s", ErrNeedsAttention, candidatePath)
	}
	if err := m.nginx.NginxReload(ctx); err != nil {
		if restoreErr := restoreExchange(configPath, candidateInfo, newConfig, candidatePath); restoreErr != nil {
			keepBackup = true
			return contract.SiteSummary{}, restoreErr
		}
		if recoveryErr := m.validateAndReloadPrevious(ctx); recoveryErr != nil {
			return contract.SiteSummary{}, recoveryErr
		}
		return contract.SiteSummary{}, fmt.Errorf("%w: Nginx reload failed and the previous configuration was restored: %v", ErrUnavailable, err)
	}
	if !fileMatches(configPath, candidateInfo, newConfig) {
		keepBackup = true
		return contract.SiteSummary{}, fmt.Errorf("%w: candidate changed while reloading; backup retained at %s", ErrNeedsAttention, candidatePath)
	}
	if err := os.Remove(candidatePath); err != nil {
		keepBackup = true
		return contract.SiteSummary{}, fmt.Errorf("%w: update succeeded but backup cleanup failed at %s: %v", ErrNeedsAttention, candidatePath, err)
	}
	if err := syncDirectory(filepath.Dir(configPath)); err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: update succeeded but configuration directory cleanup could not be synced: %v", ErrNeedsAttention, err)
	}
	return m.discoverManaged(spec.Primary)
}

type DeleteResult struct {
	ID              string `json:"id"`
	PrimaryDomain   string `json:"primaryDomain"`
	Status          string `json:"status"`
	ResourceVersion string `json:"resourceVersion"`
}

func (m *Manager) Delete(ctx context.Context, id, expectedVersion string) (DeleteResult, error) {
	if expectedVersion == "" {
		return DeleteResult{}, fmt.Errorf("%w: expectedResourceVersion is required", ErrInvalidInput)
	}
	siteWriteMutex.Lock()
	defer siteWriteMutex.Unlock()
	if err := m.Writable(ctx); err != nil {
		return DeleteResult{}, err
	}
	current, err := m.findManagedByID(id)
	if err != nil {
		return DeleteResult{}, err
	}
	if current.Kind != contract.SiteReverseProxy {
		return DeleteResult{}, fmt.Errorf("%w: only a Panel-managed reverse proxy can be deleted", ErrForbidden)
	}
	if current.ResourceVersion != expectedVersion {
		return DeleteResult{}, fmt.Errorf("%w: resource version changed", ErrConflict)
	}
	spec, err := managedSpecFromSummary(current)
	if err != nil {
		return DeleteResult{}, fmt.Errorf("%w: current site is no longer canonical", ErrForbidden)
	}
	config := renderManagedConfig(spec)
	configPath := filepath.Join(m.webRoot, "conf.d", spec.Primary+".conf")
	configInfo, err := os.Lstat(configPath)
	if err == nil && !fileMatches(configPath, configInfo, config) {
		legacyConfig := renderManagedConfigV1(spec)
		if fileMatches(configPath, configInfo, legacyConfig) {
			config = legacyConfig
		}
	}
	if err != nil || !fileMatches(configPath, configInfo, config) {
		return DeleteResult{}, fmt.Errorf("%w: current configuration is no longer canonical", ErrForbidden)
	}
	backup, err := os.CreateTemp(filepath.Dir(configPath), "."+spec.Primary+".kp-delete-*.tmp")
	if err != nil {
		return DeleteResult{}, fmt.Errorf("%w: stage delete backup: %v", ErrUnavailable, err)
	}
	backupPath := backup.Name()
	if closeErr := backup.Close(); closeErr != nil {
		_ = os.Remove(backupPath)
		return DeleteResult{}, fmt.Errorf("%w: close delete backup: %v", ErrUnavailable, closeErr)
	}
	if err := os.Remove(backupPath); err != nil {
		return DeleteResult{}, fmt.Errorf("%w: prepare delete backup: %v", ErrUnavailable, err)
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			_ = os.Remove(backupPath)
		}
	}()
	if err := os.Rename(configPath, backupPath); err != nil {
		return DeleteResult{}, fmt.Errorf("%w: atomically withdraw configuration: %v", ErrUnavailable, err)
	}
	backupInfo, _ := os.Lstat(backupPath)
	if !fileMatches(backupPath, backupInfo, config) {
		keepBackup = true
		return DeleteResult{}, fmt.Errorf("%w: delete backup changed unexpectedly at %s", ErrNeedsAttention, backupPath)
	}
	if err := syncDirectory(filepath.Dir(configPath)); err != nil {
		if restoreErr := restoreDeletedConfig(configPath, backupPath, backupInfo, config); restoreErr != nil {
			keepBackup = true
			return DeleteResult{}, restoreErr
		}
		return DeleteResult{}, fmt.Errorf("%w: sync delete transaction; configuration restored: %v", ErrUnavailable, err)
	}
	if err := m.nginx.NginxTest(ctx); err != nil {
		if restoreErr := restoreDeletedConfig(configPath, backupPath, backupInfo, config); restoreErr != nil {
			keepBackup = true
			return DeleteResult{}, restoreErr
		}
		return DeleteResult{}, fmt.Errorf("%w: delete candidate failed nginx -t: %v", ErrUnprocessable, err)
	}
	if err := m.nginx.NginxReload(ctx); err != nil {
		if restoreErr := restoreDeletedConfig(configPath, backupPath, backupInfo, config); restoreErr != nil {
			keepBackup = true
			return DeleteResult{}, restoreErr
		}
		if recoveryErr := m.validateAndReloadPrevious(ctx); recoveryErr != nil {
			return DeleteResult{}, recoveryErr
		}
		return DeleteResult{}, fmt.Errorf("%w: Nginx reload failed and the configuration was restored: %v", ErrUnavailable, err)
	}
	if _, err := os.Lstat(configPath); !errors.Is(err, os.ErrNotExist) {
		keepBackup = true
		return DeleteResult{}, fmt.Errorf("%w: configuration path was recreated during deletion; backup retained at %s", ErrNeedsAttention, backupPath)
	}
	if err := os.Remove(backupPath); err != nil {
		keepBackup = true
		return DeleteResult{}, fmt.Errorf("%w: proxy was deleted but backup cleanup failed at %s: %v", ErrNeedsAttention, backupPath, err)
	}
	if err := syncDirectory(filepath.Dir(configPath)); err != nil {
		return DeleteResult{}, fmt.Errorf("%w: proxy was deleted but directory cleanup could not be synced: %v", ErrNeedsAttention, err)
	}
	return DeleteResult{
		ID: current.ID, PrimaryDomain: current.PrimaryDomain,
		Status: "deleted", ResourceVersion: current.ResourceVersion,
	}, nil
}

func restoreDeletedConfig(configPath, backupPath string, backupInfo os.FileInfo, config []byte) error {
	if !fileMatches(backupPath, backupInfo, config) {
		return fmt.Errorf("%w: delete backup changed; retained at %s", ErrNeedsAttention, backupPath)
	}
	if _, err := os.Lstat(configPath); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: configuration path is occupied; backup retained at %s", ErrNeedsAttention, backupPath)
	}
	if err := os.Rename(backupPath, configPath); err != nil {
		return fmt.Errorf("%w: restore deleted configuration from %s: %v", ErrNeedsAttention, backupPath, err)
	}
	if err := syncDirectory(filepath.Dir(configPath)); err != nil {
		return fmt.Errorf("%w: sync restored configuration: %v", ErrNeedsAttention, err)
	}
	return nil
}

func (m *Manager) checkCollisions(spec managedSpec, excludeID string) error {
	items, err := m.discoverer.Discover()
	if err != nil {
		return fmt.Errorf("%w: discover existing sites: %v", ErrUnavailable, err)
	}
	requested := make(map[string]bool, len(spec.Aliases)+1)
	requested[spec.Primary] = true
	for _, alias := range spec.Aliases {
		requested[alias] = true
	}
	for _, site := range items {
		if site.ID == excludeID {
			continue
		}
		if requested[site.PrimaryDomain] {
			return fmt.Errorf("%w: domain %s is already configured", ErrConflict, site.PrimaryDomain)
		}
		for _, domain := range site.Domains {
			if requested[domain] {
				return fmt.Errorf("%w: domain %s is already configured", ErrConflict, domain)
			}
		}
	}
	for domain := range requested {
		conflictPaths := []string{
			filepath.Join(m.webRoot, "conf.d", domain+".conf"),
			filepath.Join(m.webRoot, "html", domain),
		}
		for _, path := range conflictPaths {
			if excludeID != "" && domain == spec.Primary &&
				(path == filepath.Join(m.webRoot, "conf.d", spec.Primary+".conf") ||
					(siteNeedsDocumentRoot(spec.Kind) && path == filepath.Join(m.webRoot, "html", spec.Primary))) {
				continue
			}
			if _, statErr := os.Lstat(path); statErr == nil {
				return fmt.Errorf("%w: managed path already exists for %s", ErrConflict, domain)
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return fmt.Errorf("%w: inspect collision path: %v", ErrUnavailable, statErr)
			}
		}
	}
	return nil
}

func (m *Manager) findManagedByID(id string) (contract.SiteSummary, error) {
	items, err := m.discoverer.Discover()
	if err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: discover sites: %v", ErrUnavailable, err)
	}
	for _, site := range items {
		if site.ID == id {
			if (site.Origin != contract.OriginWeb && site.Origin != contract.OriginCLI) ||
				!containsString(site.AllowedActions, "update") {
				return contract.SiteSummary{}, fmt.Errorf("%w: only a recognized unchanged site can be updated", ErrForbidden)
			}
			return site, nil
		}
	}
	return contract.SiteSummary{}, fmt.Errorf("%w: site does not exist", ErrConflict)
}

func (m *Manager) discoverManaged(primary string) (contract.SiteSummary, error) {
	items, err := m.discoverer.Discover()
	if err != nil {
		return contract.SiteSummary{}, fmt.Errorf("%w: rediscover committed site: %v", ErrNeedsAttention, err)
	}
	for _, site := range items {
		if site.PrimaryDomain == primary && site.Origin == contract.OriginWeb {
			return site, nil
		}
	}
	return contract.SiteSummary{}, fmt.Errorf("%w: committed site could not be rediscovered", ErrNeedsAttention)
}

func (m *Manager) rollbackCreate(
	configPath string,
	configInfo os.FileInfo,
	config []byte,
	staticPath string,
	staticInfo os.FileInfo,
	spec managedSpec,
) error {
	if !fileMatches(configPath, configInfo, config) {
		return fmt.Errorf("%w: candidate changed; refusing rollback", ErrNeedsAttention)
	}
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("%w: remove failed candidate: %v", ErrNeedsAttention, err)
	}
	if err := syncDirectory(filepath.Dir(configPath)); err != nil {
		return fmt.Errorf("%w: sync rollback: %v", ErrNeedsAttention, err)
	}
	return m.removeCreatedStatic(staticPath, staticInfo, spec)
}

func (m *Manager) removeCreatedStatic(path string, expected os.FileInfo, spec managedSpec) error {
	if path == "" {
		return nil
	}
	current, err := os.Lstat(path)
	if err != nil || expected == nil || !os.SameFile(expected, current) || !current.IsDir() {
		return fmt.Errorf("%w: document root changed; refusing rollback", ErrNeedsAttention)
	}
	indexName, indexBody := managedDefaultDocument(spec)
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 1 || entries[0].Name() != indexName ||
		entries[0].Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: document root contains external changes; refusing rollback", ErrNeedsAttention)
	}
	indexPath := filepath.Join(path, indexName)
	indexInfo, err := os.Lstat(indexPath)
	if err != nil || !fileMatches(indexPath, indexInfo, indexBody) {
		return fmt.Errorf("%w: default index changed; refusing rollback", ErrNeedsAttention)
	}
	if err := os.Remove(indexPath); err != nil {
		return fmt.Errorf("%w: remove staged index: %v", ErrNeedsAttention, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("%w: remove staged document root: %v", ErrNeedsAttention, err)
	}
	return syncDirectory(filepath.Dir(path))
}

func (m *Manager) validateAndReloadPrevious(ctx context.Context) error {
	if err := m.nginx.NginxTest(ctx); err != nil {
		return fmt.Errorf("%w: rollback completed but previous Nginx configuration is invalid: %v", ErrNeedsAttention, err)
	}
	if err := m.nginx.NginxReload(ctx); err != nil {
		return fmt.Errorf("%w: rollback completed but previous Nginx configuration could not be reloaded: %v", ErrNeedsAttention, err)
	}
	return nil
}

func (m *Manager) callHook(stage, path string) {
	if m.testHook != nil {
		m.testHook(stage, path)
	}
}

func restoreExchange(configPath string, candidateInfo os.FileInfo, candidate []byte, backupPath string) error {
	if !fileMatches(configPath, candidateInfo, candidate) {
		return fmt.Errorf("%w: candidate changed; previous configuration retained at %s", ErrNeedsAttention, backupPath)
	}
	if err := atomicExchange(configPath, backupPath); err != nil {
		return fmt.Errorf("%w: restore previous configuration from %s: %v", ErrNeedsAttention, backupPath, err)
	}
	if err := syncDirectory(filepath.Dir(configPath)); err != nil {
		return fmt.Errorf("%w: sync restored configuration: %v", ErrNeedsAttention, err)
	}
	return nil
}

func writeTemporaryFile(directory, pattern string, data []byte, mode os.FileMode) (string, error) {
	file, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return "", err
	}
	path := file.Name()
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(mode); err != nil {
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		return "", err
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := syncDirectory(directory); err != nil {
		return "", err
	}
	ok = true
	return path, nil
}

func stageStaticDirectory(parent, primary, indexName string, index []byte) (string, error) {
	path, err := os.MkdirTemp(parent, "."+primary+".kp-*")
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		if !ok {
			_ = removeStagedStaticDirectory(path, indexName)
		}
	}()
	if err := os.Chmod(path, 0o755); err != nil {
		return "", err
	}
	indexPath := filepath.Join(path, indexName)
	file, err := os.OpenFile(indexPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(index); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := syncDirectory(path); err != nil {
		return "", err
	}
	if err := syncDirectory(parent); err != nil {
		return "", err
	}
	ok = true
	return path, nil
}

func removeStagedStaticDirectory(path, indexName string) error {
	if path == "" {
		return nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(entries) != 1 || entries[0].Name() != indexName || entries[0].Type()&os.ModeSymlink != 0 {
		return errors.New("unexpected staged document root contents")
	}
	if err := os.Remove(filepath.Join(path, indexName)); err != nil {
		return err
	}
	return os.Remove(path)
}

func fileMatches(path string, expectedInfo os.FileInfo, expected []byte) bool {
	if expectedInfo == nil {
		return false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(expectedInfo, info) {
		return false
	}
	data, err := readRegularFile(path, int64(len(expected)))
	return err == nil && bytes.Equal(data, expected)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
