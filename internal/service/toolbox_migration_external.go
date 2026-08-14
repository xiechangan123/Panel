package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cast"
	"go.yaml.in/yaml/v4"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/types"
	webtypes "github.com/acepanel/panel/v3/pkg/webserver/types"
)

var errMigrationConflict = errors.New("migration target conflict")

type preparedMigrationItem struct {
	detail   *types.MigrationSourceDetail
	artifact *types.MigrationArtifact
}

func (s *ToolboxMigrationService) prepareMigrationCatalog(items []types.MigrationSourceItem) {
	ctx := context.Background()
	websitePath, _ := s.settingRepo.Get(biz.SettingKeyWebsitePath, filepath.Join(app.Root, "sites"))
	projectPath, _ := s.settingRepo.Get(biz.SettingKeyProjectPath, filepath.Join(app.Root, "projects"))
	projects, _, _ := s.projectRepo.List("", 1, 10000)
	containers, dockerErr := s.containerRepo.ListAll()
	composes, composeErr := s.composeRepo.List()
	databases, _, _ := s.databaseRepo.List(ctx, 1, 10000, "")
	servers, _, _ := s.databaseServerRepo.List(ctx, 1, 10000, "")
	projectNames := lo.SliceToMap(projects, func(project *types.ProjectDetail) (string, struct{}) {
		return project.Name, struct{}{}
	})
	containerNames := lo.SliceToMap(containers, func(container types.Container) (string, struct{}) {
		return container.Name, struct{}{}
	})
	composeNames := lo.SliceToMap(composes, func(compose types.ContainerCompose) (string, struct{}) {
		return compose.Name, struct{}{}
	})
	serverNames := lo.SliceToMap(servers, func(server *biz.DatabaseServer) (string, struct{}) {
		return server.Name, struct{}{}
	})
	databaseNames := lo.SliceToMap(databases, func(database *biz.Database) (string, struct{}) {
		return database.Server + "\x00" + database.Name, struct{}{}
	})
	resourceNamePattern := regexp.MustCompile(`^([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9._-]{0,126}[A-Za-z0-9_-])$`)
	mysqlNamePattern := regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	databaseNamePattern := regexp.MustCompile(`^[A-Za-z0-9_.-]{1,63}$`)

	for i := range items {
		item := &items[i]
		switch item.Type {
		case "website":
			item.TargetPath = filepath.Join(websitePath, item.TargetName, "public")
			if _, err := s.websiteRepo.GetByName(item.TargetName); err == nil {
				item.Blockers = append(item.Blockers, s.t.Get("a website with the same name already exists on the target server"))
			}
		case "database":
			targetType := item.Subtype
			if targetType == "mariadb" {
				targetType = "mysql"
			}
			localServer := "local_" + targetType
			if _, exists := serverNames[localServer]; !exists {
				item.Blockers = append(item.Blockers, s.t.Get("the target server does not have a compatible %s database server", item.Subtype))
			}
			if _, exists := databaseNames[localServer+"\x00"+item.TargetName]; exists {
				item.Blockers = append(item.Blockers, s.t.Get("a database with the same name already exists on the target server"))
			}
		case "project":
			item.TargetPath = filepath.Join(projectPath, item.TargetName)
			if _, exists := projectNames[item.TargetName]; exists {
				item.Blockers = append(item.Blockers, s.t.Get("a project with the same name already exists on the target server"))
			}
			if strings.HasPrefix(item.Subtype, "runtime_") || item.Subtype == "appstore" {
				if dockerErr != nil || composeErr != nil {
					item.Blockers = append(item.Blockers, s.t.Get("Docker Compose is not available on the target server"))
				}
			} else if item.Subtype != "general" && item.Subtype != "" {
				if _, compatible := s.compatibleProjectRuntime(types.ProjectType(item.Subtype), item.RuntimeVersion); !compatible {
					item.Blockers = append(item.Blockers, s.t.Get(
						"the target server does not have a compatible %s runtime for version %s",
						item.Subtype,
						lo.CoalesceOrEmpty(item.RuntimeVersion, s.t.Get("unknown")),
					))
				}
			}
		case "container":
			if dockerErr != nil {
				item.Blockers = append(item.Blockers, s.t.Get("Docker is not available on the target server"))
			}
			if _, exists := containerNames[item.TargetName]; exists {
				item.Blockers = append(item.Blockers, s.t.Get("a container with the same name already exists on the target server"))
			}
		case "compose":
			item.TargetPath = filepath.Join(app.Root, "compose", item.TargetName)
			if dockerErr != nil || composeErr != nil {
				item.Blockers = append(item.Blockers, s.t.Get("Docker Compose is not available on the target server"))
			}
			if _, exists := composeNames[item.TargetName]; exists {
				item.Blockers = append(item.Blockers, s.t.Get("a Compose project with the same name already exists on the target server"))
			}
		}

		validName := true
		switch item.Type {
		case "website":
			validName = resourceNamePattern.MatchString(item.TargetName)
			validName = validName && item.TargetName != "default" && item.TargetName != "phpmyadmin"
		case "database":
			if item.Subtype == "mysql" || item.Subtype == "mariadb" {
				validName = mysqlNamePattern.MatchString(item.TargetName)
			} else {
				validName = databaseNamePattern.MatchString(item.TargetName)
			}
		case "project", "container", "compose":
			validName = resourceNamePattern.MatchString(item.TargetName)
		}
		if !validName {
			item.Blockers = append(item.Blockers, s.t.Get("the resource name is not supported by AcePanel"))
		}
	}
}

func (s *ToolboxMigrationService) runExternalMigration(conn *request.ToolboxMigrationConnection, req *request.ToolboxMigrationItems) {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	s.addLog("===== " + s.t.Get("Migration started") + " =====")
	defer func() {
		s.state.mu.Lock()
		s.state.Step = types.MigrationStepDone
		s.state.EndedAt = new(time.Now())
		s.state.mu.Unlock()
		s.addLog("===== " + s.t.Get("Migration completed") + " =====")
	}()

	catalog, err := s.sourceRepo.Items(ctx, conn)
	if err != nil {
		s.addLog(s.t.Get("failed to refresh source resource list: %v", err))
		return
	}
	s.prepareMigrationCatalog(catalog)
	byKey := lo.SliceToMap(catalog, func(item types.MigrationSourceItem) (string, types.MigrationSourceItem) {
		return item.Key, item
	})
	selectedKeys := lo.SliceToMap(req.Items, func(item request.ToolboxMigrationSelectedItem) (string, struct{}) {
		return item.Key, struct{}{}
	})

	prepared := make([]preparedMigrationItem, 0, len(req.Items))
	for _, selected := range req.Items {
		item, ok := byKey[selected.Key]
		if !ok {
			s.addResult(types.MigrationItemResult{
				Key: selected.Key, Type: "unknown", Name: selected.Key,
				Status: types.MigrationItemFailed, Stage: types.MigrationStageDone,
				Error: s.t.Get("resource no longer exists on the source server"),
			})
			continue
		}
		result := types.MigrationItemResult{
			Key: item.Key, Type: item.Type, Name: item.Name,
			Status: types.MigrationItemPending, Stage: types.MigrationStagePreparing,
			Warnings: append([]string(nil), item.Warnings...),
		}
		s.addResult(result)
		if !item.Supported || len(item.Blockers) > 0 {
			reason := strings.Join(item.Blockers, "; ")
			if reason == "" {
				reason = s.t.Get("this resource type is not supported")
			}
			if req.SkipIncompatibleItems {
				s.completeExternalResult(item.Key, types.MigrationItemSkipped, reason, nil, nil)
			} else {
				s.failExternalResult(item.Key, reason, nil)
			}
			continue
		}
		if _, missing := lo.Find(item.DependsOn, func(dependency string) bool {
			_, selected := selectedKeys[dependency]
			return !selected
		}); missing {
			s.failExternalResult(item.Key, s.t.Get("a required dependency was not selected"), nil)
			continue
		}
		detail, detailErr := s.sourceRepo.Detail(ctx, conn, item)
		if detailErr != nil {
			s.failExternalResult(item.Key, s.t.Get("failed to refresh source resource detail: %v", detailErr), nil)
			continue
		}
		detail.Item = item
		if selected.TargetPath != "" {
			detail.Item.TargetPath = selected.TargetPath
		}
		if selected.TargetUser != "" && detail.Project != nil {
			detail.Project.User = selected.TargetUser
		}
		prepared = append(prepared, preparedMigrationItem{detail: detail})
	}

	priorities := map[string]int{
		"database": 1, "database_user": 1, "container": 2,
		"compose": 3, "project": 4, "website": 5,
	}
	slices.SortStableFunc(prepared, func(a, b preparedMigrationItem) int {
		return priorities[a.detail.Item.Type] - priorities[b.detail.Item.Type]
	})

	stopped := make([]*types.MigrationSourceDetail, 0)
	restoreSource := func() {
		restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer restoreCancel()
		for _, detail := range slices.Backward(stopped) {
			if restoreErr := s.sourceRepo.SetRunning(restoreCtx, conn, detail, true); restoreErr != nil {
				s.addExternalWarning(detail.Item.Key, s.t.Get("failed to restore source resource: %v", restoreErr))
			}
		}
		stopped = nil
	}
	defer restoreSource()
	if req.StopSourceDuringBackup {
		for i := range prepared {
			detail := prepared[i].detail
			if detail.Item.Status != "running" || detail.Item.Type == "database" {
				continue
			}
			if stopErr := s.sourceRepo.SetRunning(ctx, conn, detail, false); stopErr != nil {
				s.addExternalWarning(detail.Item.Key, s.t.Get("failed to stop source resource before backup: %v", stopErr))
				continue
			}
			stopped = append(stopped, detail)
		}
	}

	for i := range prepared {
		item := &prepared[i]
		s.setExternalResult(item.detail.Item.Key, types.MigrationItemRunning, types.MigrationStagePreparing)
		artifact, prepareErr := s.sourceRepo.Prepare(ctx, conn, item.detail)
		if prepareErr != nil {
			s.failExternalResult(item.detail.Item.Key, s.t.Get("failed to create source backup: %v", prepareErr), nil)
			continue
		}
		item.artifact = artifact
	}
	restoreSource()

	for i := range prepared {
		item := &prepared[i]
		s.state.mu.RLock()
		results := lo.SliceToMap(s.state.Results, func(result types.MigrationItemResult) (string, types.MigrationItemResult) {
			return result.Key, result
		})
		s.state.mu.RUnlock()
		result := results[item.detail.Item.Key]
		finished := result.Status == types.MigrationItemFailed || result.Status == types.MigrationItemSkipped
		if item.artifact == nil || finished {
			continue
		}
		dependencyFailed := lo.SomeBy(item.detail.Item.DependsOn, func(dependency string) bool {
			result, exists := results[dependency]
			return !exists || result.Status != types.MigrationItemSuccess && result.Status != types.MigrationItemPartial
		})
		if dependencyFailed {
			reason := s.t.Get("a required dependency did not migrate successfully")
			if req.SkipIncompatibleItems {
				s.completeExternalResult(item.detail.Item.Key, types.MigrationItemSkipped, reason, nil, nil)
			} else {
				s.failExternalResult(item.detail.Item.Key, reason, nil)
			}
			continue
		}
		tmpDir, tmpErr := s.migrationTmpDir()
		if tmpErr != nil {
			s.failExternalResult(item.detail.Item.Key, tmpErr.Error(), nil)
			continue
		}
		artifactPath := filepath.Join(tmpDir, item.artifact.FileName)
		if strings.Contains(item.artifact.Kind, "bundle") {
			artifactPath += ".bundle.tar.gz"
		}
		s.setExternalResult(item.detail.Item.Key, types.MigrationItemRunning, types.MigrationStageDownloading)
		downloadErr := s.sourceRepo.Download(ctx, conn, item.artifact, artifactPath)
		if downloadErr != nil {
			_ = os.RemoveAll(tmpDir)
			s.failExternalResult(item.detail.Item.Key, s.t.Get("failed to download source backup: %v", downloadErr), nil)
			continue
		}
		s.setExternalResult(item.detail.Item.Key, types.MigrationItemRunning, types.MigrationStageValidating)
		s.setExternalResult(item.detail.Item.Key, types.MigrationItemRunning, types.MigrationStageImporting)
		var created, warnings []string
		var importErr error
		switch item.detail.Item.Type {
		case "database":
			created, warnings, importErr = s.importExternalDatabase(ctx, item.detail, artifactPath)
		case "website":
			created, warnings, importErr = s.importExternalWebsite(ctx, item.detail, artifactPath)
		case "project":
			if item.detail.Compose != nil && item.detail.Compose.Compose != "" {
				created, warnings, importErr = s.importExternalCompose(ctx, item.detail, artifactPath)
			} else {
				created, warnings, importErr = s.importExternalProject(ctx, item.detail, artifactPath)
			}
		case "container":
			created, warnings, importErr = s.importExternalContainer(ctx, item.detail, artifactPath)
		case "compose":
			created, warnings, importErr = s.importExternalCompose(ctx, item.detail, artifactPath)
		default:
			importErr = errors.New(s.t.Get("unsupported migration resource type: %s", item.detail.Item.Type))
		}
		s.state.mu.RLock()
		results = lo.SliceToMap(s.state.Results, func(result types.MigrationItemResult) (string, types.MigrationItemResult) {
			return result.Key, result
		})
		s.state.mu.RUnlock()
		for _, dependency := range item.detail.Item.DependsOn {
			result, exists := results[dependency]
			if !exists || result.Status != types.MigrationItemPartial {
				continue
			}
			for _, warning := range result.Warnings {
				warnings = append(warnings, s.t.Get("dependency %s: %s", result.Name, warning))
			}
		}
		_ = os.RemoveAll(tmpDir)
		if errors.Is(importErr, errMigrationConflict) {
			s.completeExternalResult(item.detail.Item.Key, types.MigrationItemSkipped, importErr.Error(), nil, nil)
			continue
		}
		if importErr != nil {
			s.failExternalResult(item.detail.Item.Key, importErr.Error(), created)
			continue
		}
		if len(warnings) > 0 {
			s.completeExternalResult(item.detail.Item.Key, types.MigrationItemPartial, "", created, warnings)
		} else {
			s.completeExternalResult(item.detail.Item.Key, types.MigrationItemSuccess, "", created, nil)
		}
	}

}

func (s *ToolboxMigrationService) setExternalResult(key string, status types.MigrationItemStatus, stage types.MigrationItemStage) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	index := slices.IndexFunc(s.state.Results, func(result types.MigrationItemResult) bool { return result.Key == key })
	if index < 0 {
		return
	}
	result := &s.state.Results[index]
	result.Status = status
	result.Stage = stage
	if result.StartedAt == nil && status == types.MigrationItemRunning {
		result.StartedAt = new(time.Now())
	}
}

func (s *ToolboxMigrationService) addExternalWarning(key, warning string) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	index := slices.IndexFunc(s.state.Results, func(result types.MigrationItemResult) bool { return result.Key == key })
	if index >= 0 {
		s.state.Results[index].Warnings = append(s.state.Results[index].Warnings, warning)
	}
}

func (s *ToolboxMigrationService) completeExternalResult(key string, status types.MigrationItemStatus, message string, created, warnings []string) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	index := slices.IndexFunc(s.state.Results, func(result types.MigrationItemResult) bool { return result.Key == key })
	if index < 0 {
		return
	}
	result := &s.state.Results[index]
	now := time.Now()
	result.Status = status
	result.Stage = types.MigrationStageDone
	result.Error = message
	result.Created = append(result.Created, created...)
	result.Warnings = append(result.Warnings, warnings...)
	result.EndedAt = &now
	if result.StartedAt != nil {
		result.Duration = now.Sub(*result.StartedAt).Seconds()
	}
}

func (s *ToolboxMigrationService) failExternalResult(key, message string, residuals []string) {
	s.completeExternalResult(key, types.MigrationItemFailed, message, nil, nil)
	if len(residuals) > 0 {
		s.state.mu.Lock()
		index := slices.IndexFunc(s.state.Results, func(result types.MigrationItemResult) bool { return result.Key == key })
		if index >= 0 {
			s.state.Results[index].Residuals = append(s.state.Results[index].Residuals, residuals...)
		}
		s.state.mu.Unlock()
	}
	s.addLog(s.t.Get("migration failed: %s", message))
}

func (s *ToolboxMigrationService) importExternalDatabase(
	ctx context.Context,
	detail *types.MigrationSourceDetail,
	artifactPath string,
) ([]string, []string, error) {
	database := detail.Database
	if database == nil {
		return nil, nil, errors.New(s.t.Get("database detail is missing"))
	}
	targetType := database.Type
	if targetType == "mariadb" {
		targetType = "mysql"
	}
	server, err := s.databaseServerRepo.GetByName(ctx, "local_"+targetType)
	if err != nil {
		return nil, nil, errors.New(s.t.Get("no compatible database server is installed on the target"))
	}
	existing, _, err := s.databaseRepo.List(ctx, 1, 10000, targetType)
	if err != nil {
		return nil, nil, err
	}
	if slices.ContainsFunc(existing, func(item *biz.Database) bool { return item.ServerID == server.ID && item.Name == database.Name }) {
		return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target database already exists"))
	}
	if database.Username != "" {
		users, _, userErr := s.databaseUserRepo.List(ctx, 1, 10000, targetType)
		if userErr != nil {
			return nil, nil, userErr
		}
		if slices.ContainsFunc(users, func(item *biz.DatabaseUser) bool {
			return item.ServerID == server.ID && item.Username == database.Username
		}) {
			return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target database user already exists"))
		}
	}
	if targetType == "postgresql" {
		installed, installedErr := s.appRepo.GetInstalled("postgresql")
		versionPattern := regexp.MustCompile(`\d+`)
		if installedErr == nil && cast.ToInt(versionPattern.FindString(database.Version)) > cast.ToInt(versionPattern.FindString(installed.Version)) {
			return nil, nil, errors.New(s.t.Get("the source PostgreSQL major version is newer than the target version"))
		}
	}

	create := &request.DatabaseCreate{ServerID: server.ID, Name: database.Name}
	warnings := make([]string, 0)
	if database.Username != "" && database.PasswordOK {
		create.CreateUser = true
		create.Username = database.Username
		create.Password = database.Password
		create.Host = lo.CoalesceOrEmpty(database.Host, "localhost")
	} else if database.Username != "" {
		warnings = append(
			warnings,
			s.t.Get("the database was imported but the source password was unavailable; reset the database user password and update the website configuration manually"),
		)
	}
	if database.Type != targetType {
		warnings = append(warnings, s.t.Get("the database is being imported across MySQL and MariaDB; verify application compatibility after migration"))
	}
	if err = s.databaseRepo.Create(ctx, create); err != nil {
		return nil, warnings, err
	}
	created := []string{s.t.Get("database: %s", database.Name)}

	var backupType biz.BackupType
	switch targetType {
	case "mysql":
		backupType = biz.BackupTypeMySQL
	case "postgresql":
		backupType = biz.BackupTypePostgres
	default:
		return created, warnings, errors.New(s.t.Get("unsupported database type: %s", targetType))
	}
	if err = s.backupRepo.Restore(ctx, backupType, artifactPath, database.Name); err != nil {
		return created, warnings, errors.New(s.t.Get("database import failed: %v", err))
	}
	return created, warnings, nil
}

func (s *ToolboxMigrationService) importExternalWebsite(
	ctx context.Context,
	detail *types.MigrationSourceDetail,
	artifactPath string,
) ([]string, []string, error) {
	website := detail.Website
	if website == nil {
		return nil, nil, errors.New(s.t.Get("website detail is missing"))
	}
	if _, err := s.websiteRepo.GetByName(detail.Item.TargetName); err == nil {
		return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target website already exists"))
	}
	websitePath, _ := s.settingRepo.Get(biz.SettingKeyWebsitePath, filepath.Join(app.Root, "sites"))
	targetPath := filepath.Join(websitePath, detail.Item.TargetName, "public")
	if entries, readErr := os.ReadDir(targetPath); readErr == nil && len(entries) > 0 {
		return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target website directory is not empty"))
	}
	listens := make([]string, 0, len(website.Listens))
	for _, value := range website.Listens {
		sslListen := slices.Contains(website.SSLListens, value)
		if len(website.SSLListens) == 0 {
			sslListen = value == "443" || strings.HasSuffix(value, ":443")
		}
		if value != "" && !sslListen {
			listens = append(listens, value)
		}
	}
	if len(listens) == 0 {
		listens = []string{"80"}
	}
	domains := slices.DeleteFunc(append([]string(nil), website.Domains...), func(domain string) bool { return domain == "" })
	if len(domains) == 0 {
		domains = []string{detail.Item.Name}
	}
	create := &request.WebsiteCreate{
		Type: website.Type, Name: detail.Item.TargetName, Listens: listens, Domains: domains, Path: targetPath,
		PHP: website.PHP,
	}
	if create.Type != "php" && create.Type != "proxy" && create.Type != "static" {
		create.Type = "static"
	}
	if create.Type == "php" && create.PHP == 0 {
		return nil, nil, errors.New(s.t.Get("the source PHP version could not be determined"))
	}
	if create.Type == "php" && !slices.Contains(s.environmentRepo.InstalledSlugs("php"), strconv.FormatUint(uint64(create.PHP), 10)) {
		return nil, nil, errors.New(s.t.Get("the target server does not have PHP %d installed", create.PHP))
	}
	if create.Type == "proxy" {
		if len(website.Proxies) == 0 || website.Proxies[0].Pass == "" {
			return nil, nil, errors.New(s.t.Get("the source reverse proxy target could not be determined"))
		}
		create.Proxy = website.Proxies[0].Pass
	}
	createdWebsite, err := s.websiteRepo.Create(ctx, create)
	if err != nil {
		return nil, nil, err
	}
	created := []string{s.t.Get("website: %s", detail.Item.TargetName)}
	staging, err := os.MkdirTemp(filepath.Dir(targetPath), ".website-migration-*")
	if err != nil {
		return created, nil, errors.New(s.t.Get("website file import failed: %v", err))
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err = s.extractTar(ctx, artifactPath, staging); err != nil {
		return created, nil, errors.New(s.t.Get("website file import failed: %v", err))
	}
	inner := ""
	_ = filepath.WalkDir(staging, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr == nil && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".web.tar.gz") {
			inner = path
			return filepath.SkipAll
		}
		return nil
	})
	var contentRoot string
	if inner != "" {
		payload, payloadErr := os.MkdirTemp(staging, "payload-*")
		if payloadErr != nil {
			return created, nil, errors.New(s.t.Get("website file import failed: %v", payloadErr))
		}
		if payloadErr = s.extractTar(ctx, inner, payload); payloadErr != nil {
			return created, nil, errors.New(s.t.Get("website file import failed: %v", payloadErr))
		}
		contentRoot = s.archiveContentRoot(payload)
		if info, statErr := os.Stat(filepath.Join(contentRoot, "index")); statErr == nil && info.IsDir() {
			contentRoot = filepath.Join(contentRoot, "index")
		}
	} else {
		contentRoot = s.archiveContentRoot(staging)
	}
	if err = os.MkdirAll(targetPath, 0755); err == nil {
		err = s.copyMigrationTree(ctx, contentRoot, targetPath)
	}
	if err != nil {
		return created, nil, errors.New(s.t.Get("website file import failed: %v", err))
	}

	s.setExternalResult(detail.Item.Key, types.MigrationItemRunning, types.MigrationStageConfiguring)
	updateListens := make([]webtypes.Listen, 0, len(website.Listens))
	for _, value := range website.Listens {
		if value == "" {
			continue
		}
		listen := webtypes.Listen{Address: value}
		sslListen := slices.Contains(website.SSLListens, value)
		if len(website.SSLListens) == 0 {
			sslListen = value == "443" || strings.HasSuffix(value, ":443")
		}
		if website.SSL && sslListen {
			listen.Args = []string{"ssl"}
		}
		updateListens = append(updateListens, listen)
	}
	if website.SSL && !slices.ContainsFunc(updateListens, func(item webtypes.Listen) bool {
		return slices.Contains(item.Args, "ssl")
	}) {
		updateListens = append(updateListens, webtypes.Listen{Address: "443", Args: []string{"ssl"}})
	}
	if len(updateListens) == 0 {
		updateListens = []webtypes.Listen{{Address: "80"}}
	}
	update := &request.WebsiteUpdate{
		ID: createdWebsite.ID, Listens: updateListens, Domains: domains,
		Path: targetPath, Root: s.relocateMigrationPath(website.Root, website.Path, targetPath), Index: website.Index,
		SSL: website.SSL && website.SSLCert != "" && website.SSLKey != "", SSLCert: website.SSLCert, SSLKey: website.SSLKey,
		HSTS: website.HSTS, OCSP: website.OCSP, HTTPRedirect: website.HTTPRedirect, SSLProtocols: website.SSLProtocols,
		PHP: website.PHP, Rewrite: website.Rewrite, OpenBasedir: website.OpenBasedir,
	}
	if len(update.Index) == 0 {
		update.Index = []string{"index.php", "index.html", "index.htm"}
	}
	for _, proxy := range website.Proxies {
		update.Proxies = append(update.Proxies, webtypes.Proxy{
			Location: lo.CoalesceOrEmpty(proxy.Location, "/"), Pass: proxy.Pass, Host: proxy.Host,
			SNI: proxy.SNI, HTTPVersion: lo.CoalesceOrEmpty(proxy.HTTPVersion, "1.1"),
			Headers: proxy.Headers, Replaces: proxy.Replaces,
		})
	}
	for _, redirect := range website.Redirects {
		update.Redirects = append(update.Redirects, webtypes.Redirect{
			Type: webtypes.RedirectType(redirect.Type), From: redirect.From, To: redirect.To,
			KeepURI: redirect.KeepURI, StatusCode: redirect.StatusCode,
		})
	}
	if err = s.websiteRepo.Update(ctx, update); err != nil {
		return created, nil, errors.New(s.t.Get("website configuration import failed: %v", err))
	}
	if website.Remark != "" {
		_ = s.websiteRepo.UpdateRemark(createdWebsite.ID, website.Remark)
	}
	if website.ExpireAt != nil {
		_ = s.websiteRepo.UpdateExpireAt(createdWebsite.ID, website.ExpireAt)
	}
	dependencyPartial := false
	s.state.mu.RLock()
	for _, dependency := range detail.Item.DependsOn {
		if slices.ContainsFunc(s.state.Results, func(result types.MigrationItemResult) bool {
			return result.Key == dependency && result.Status == types.MigrationItemPartial
		}) {
			dependencyPartial = true
			break
		}
	}
	s.state.mu.RUnlock()
	if !website.Enabled || dependencyPartial {
		_ = s.websiteRepo.UpdateStatus(createdWebsite.ID, false)
	}
	return created, nil, nil
}

func (s *ToolboxMigrationService) importExternalProject(
	ctx context.Context,
	detail *types.MigrationSourceDetail,
	artifactPath string,
) ([]string, []string, error) {
	project := detail.Project
	if project == nil {
		return nil, nil, errors.New(s.t.Get("project detail is missing"))
	}
	if strings.TrimSpace(project.ExecStart) == "" {
		return nil, nil, errors.New(s.t.Get("the source project start command is missing"))
	}
	runtimeSlug, compatible := s.compatibleProjectRuntime(project.Type, project.Version)
	if !compatible {
		return nil, nil, errors.New(s.t.Get(
			"the target server does not have a compatible %s runtime for version %s",
			project.Type,
			lo.CoalesceOrEmpty(project.Version, s.t.Get("unknown")),
		))
	}
	runUser := project.User
	if runUser == "" || runUser == "www-data" || runUser == "nginx" || runUser == "apache" {
		runUser = "www"
	} else if runUser != "www" && runUser != "root" {
		if _, err := user.Lookup(runUser); err != nil {
			return nil, nil, fmt.Errorf("target user %s does not exist", runUser)
		}
	}
	projects, _, err := s.projectRepo.List("", 1, 10000)
	if err != nil {
		return nil, nil, err
	}
	if slices.ContainsFunc(projects, func(item *types.ProjectDetail) bool { return item.Name == detail.Item.TargetName }) {
		return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target project already exists"))
	}
	targetPath := detail.Item.TargetPath
	if targetPath == "" {
		root, _ := s.settingRepo.Get(biz.SettingKeyProjectPath, filepath.Join(app.Root, "projects"))
		targetPath = filepath.Join(root, detail.Item.TargetName)
	}
	if info, statErr := os.Stat(targetPath); statErr == nil && info.IsDir() {
		entries, _ := os.ReadDir(targetPath)
		if len(entries) > 0 {
			return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target project directory is not empty"))
		}
	}
	if err = os.MkdirAll(targetPath, 0755); err != nil {
		return nil, nil, err
	}
	staging, err := os.MkdirTemp(filepath.Dir(targetPath), ".project-migration-*")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err = s.extractTar(ctx, artifactPath, staging); err != nil {
		return nil, nil, err
	}
	projectRoot := s.archiveContentRoot(staging)
	s.removeMigrationProjectDependencies(projectRoot, project.Type)
	if err = s.copyMigrationTree(ctx, projectRoot, targetPath); err != nil {
		return nil, nil, err
	}

	workingDir := s.relocateMigrationPath(project.WorkingDir, project.Path, targetPath)
	execStart := strings.ReplaceAll(project.ExecStart, project.Path, targetPath)
	if runtimeSlug != "" {
		trimmed := strings.TrimLeft(execStart, " \t")
		leading := execStart[:len(execStart)-len(trimmed)]
		separator := strings.IndexAny(trimmed, " \t")
		first := trimmed
		rest := ""
		if separator >= 0 {
			first = trimmed[:separator]
			rest = trimmed[separator:]
		}
		base := filepath.Base(strings.Trim(first, "\"'"))
		executable := ""
		switch project.Type {
		case types.ProjectTypeGo:
			if base == "go" {
				executable = "go" + runtimeSlug
			}
		case types.ProjectTypeJava:
			if base == "java" {
				executable = "java" + runtimeSlug
			}
		case types.ProjectTypeNodejs:
			switch base {
			case "node":
				executable = "node" + runtimeSlug
			case "npm", "npx", "corepack":
				executable = filepath.Join(app.Root, "server", "nodejs", runtimeSlug, "bin", base)
			}
		case types.ProjectTypePython:
			if base == "python" || base == "python3" {
				executable = "python" + runtimeSlug
			}
		case types.ProjectTypeDotnet:
			if base == "dotnet" {
				executable = "dotnet" + runtimeSlug
			}
		case types.ProjectTypePHP:
			if base == "php" {
				executable = "php" + runtimeSlug
			}
		}
		if executable != "" {
			execStart = leading + executable + rest
		}
	}
	create := &request.ProjectCreate{
		Name: detail.Item.TargetName, Type: project.Type, Description: s.t.Get("Migrated from %s", detail.Item.Name),
		RootDir: targetPath, WorkingDir: workingDir, ExecStart: execStart, User: runUser,
		Restart: project.Restart, Environments: project.Environments,
	}
	if _, err = s.projectRepo.Create(ctx, create); err != nil {
		return nil, nil, err
	}
	created := []string{s.t.Get("project: %s", detail.Item.TargetName)}
	warnings := make([]string, 0)
	if project.Port > 0 && len(project.Domains) > 0 {
		if _, websiteErr := s.websiteRepo.GetByName(detail.Item.TargetName); websiteErr == nil {
			warnings = append(warnings, s.t.Get("the project was migrated, but its reverse proxy website already exists on the target"))
		} else {
			listens := project.Listens
			if len(listens) == 0 {
				listens = []string{"80"}
			}
			websiteRoot, _ := s.settingRepo.Get(biz.SettingKeyWebsitePath, filepath.Join(app.Root, "sites"))
			website, websiteErr := s.websiteRepo.Create(ctx, &request.WebsiteCreate{
				Type: "proxy", Name: detail.Item.TargetName, Listens: listens, Domains: project.Domains,
				Path:  filepath.Join(websiteRoot, detail.Item.TargetName, "public"),
				Proxy: "http://127.0.0.1:" + strconv.FormatUint(uint64(project.Port), 10),
			})
			if websiteErr != nil {
				warnings = append(warnings, s.t.Get("the project was migrated, but its reverse proxy website could not be created: %v", websiteErr))
			} else {
				created = append(created, s.t.Get("website: %s", detail.Item.TargetName))
				if !project.Running || project.Type == types.ProjectTypeNodejs || project.Type == types.ProjectTypePython || project.Type == types.ProjectTypeGeneral {
					_ = s.websiteRepo.UpdateStatus(website.ID, false)
				}
			}
		}
	}
	switch project.Type {
	case types.ProjectTypeNodejs:
		warnings = append(warnings, s.t.Get("Node.js dependencies were not installed; install dependencies and start the project manually"))
	case types.ProjectTypePython:
		warnings = append(
			warnings,
			s.t.Get("the source Python virtual environment was not migrated; create an environment, install dependencies, and start the project manually"),
		)
	case types.ProjectTypeGeneral:
		warnings = append(warnings, s.t.Get("the general project was restored but left stopped for manual verification"))
	default:
		if project.Enabled {
			if _, enableErr := shell.Exec("systemctl enable " + strconv.Quote(detail.Item.TargetName)); enableErr != nil {
				warnings = append(warnings, s.t.Get("the project was restored but could not be enabled at startup: %v", enableErr))
			}
		}
		if project.Running {
			s.setExternalResult(detail.Item.Key, types.MigrationItemRunning, types.MigrationStageStarting)
			if _, startErr := shell.Exec("systemctl start " + strconv.Quote(detail.Item.TargetName)); startErr != nil {
				warnings = append(warnings, s.t.Get("the project was restored but could not be started: %v", startErr))
			}
		}
	}
	return created, warnings, nil
}

func (s *ToolboxMigrationService) compatibleProjectRuntime(typ types.ProjectType, sourceVersion string) (string, bool) {
	if typ == types.ProjectTypeGeneral || typ == "" {
		return "", true
	}
	if typ == types.ProjectTypeGo && strings.TrimSpace(sourceVersion) == "" {
		return "", true
	}
	runtimeType := string(typ)
	installed := s.environmentRepo.InstalledSlugs(runtimeType)
	if len(installed) == 0 {
		return "", false
	}
	if strings.TrimSpace(sourceVersion) == "" {
		return installed[0], true
	}
	versionPattern := regexp.MustCompile(`\d+`)
	versionParts := func(version string) []int {
		return lo.Map(versionPattern.FindAllString(version, 3), func(part string, _ int) int {
			return cast.ToInt(part)
		})
	}
	sourceParts := versionParts(sourceVersion)
	if typ == types.ProjectTypeJava && len(sourceParts) > 1 && sourceParts[0] == 1 {
		sourceParts = sourceParts[1:]
	}
	comparedParts := 2
	if typ == types.ProjectTypeNodejs || typ == types.ProjectTypeJava {
		comparedParts = 1
	}
	if len(sourceParts) < comparedParts {
		return "", false
	}
	for _, slug := range installed {
		targetParts := versionParts(s.environmentRepo.InstalledVersion(runtimeType, slug))
		if typ == types.ProjectTypeJava && len(targetParts) > 1 && targetParts[0] == 1 {
			targetParts = targetParts[1:]
		}
		if len(targetParts) >= comparedParts && slices.Equal(sourceParts[:comparedParts], targetParts[:comparedParts]) {
			return slug, true
		}
	}
	return "", false
}

func (s *ToolboxMigrationService) removeMigrationProjectDependencies(root string, typ types.ProjectType) {
	if typ != types.ProjectTypeNodejs && typ != types.ProjectTypePython {
		return
	}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || path == root {
			return nil
		}
		name := entry.Name()
		remove := typ == types.ProjectTypeNodejs && name == "node_modules"
		remove = remove || typ == types.ProjectTypePython && (name == ".venv" || name == "venv" || name == "virtualenv")
		if !remove {
			return nil
		}
		_ = os.RemoveAll(path)
		return filepath.SkipDir
	})
}

func (s *ToolboxMigrationService) relocateMigrationPath(value, sourceRoot, targetRoot string) string {
	if value == "" || value == sourceRoot {
		return targetRoot
	}
	if sourceRoot != "" && strings.HasPrefix(value, sourceRoot+string(os.PathSeparator)) {
		return filepath.Join(targetRoot, strings.TrimPrefix(value, sourceRoot+string(os.PathSeparator)))
	}
	return targetRoot
}

func (s *ToolboxMigrationService) importExternalContainer(
	ctx context.Context,
	detail *types.MigrationSourceDetail,
	artifactPath string,
) ([]string, []string, error) {
	container := detail.Container
	if container == nil {
		return nil, nil, errors.New(s.t.Get("container detail is missing"))
	}
	existing, err := s.containerRepo.ListAll()
	if err != nil {
		return nil, nil, err
	}
	if slices.ContainsFunc(existing, func(item types.Container) bool { return item.Name == detail.Item.TargetName }) {
		return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target container already exists"))
	}

	staging, err := os.MkdirTemp(filepath.Dir(artifactPath), ".container-import-*")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	resourceRoot := ""
	imageArchive := ""
	mountArchives := make(map[string]string)
	if strings.HasSuffix(artifactPath, ".bundle.tar.gz") {
		if err = s.extractTar(ctx, artifactPath, staging); err != nil {
			return nil, nil, err
		}
		_ = filepath.WalkDir(staging, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(entry.Name(), ".tar") && s.dockerImageArchive(ctx, path) {
				imageArchive = path
			}
			if strings.HasSuffix(entry.Name(), ".tar.gz") && !strings.Contains(entry.Name(), "bundle") {
				resourceRoot = path
				mountArchives[entry.Name()] = path
			}
			return nil
		})
	} else if s.dockerImageArchive(ctx, artifactPath) {
		imageArchive = artifactPath
	} else {
		resourceRoot = artifactPath
	}
	for i := range container.Volumes {
		remotePath := container.Volumes[i].BackupPath
		archive := mountArchives[filepath.Base(remotePath)]
		if archive == "" {
			for name, candidate := range mountArchives {
				if strings.HasSuffix(name, filepath.Base(remotePath)) {
					archive = candidate
					break
				}
			}
		}
		if remotePath == "" || archive == "" {
			continue
		}
		mountDir := filepath.Join(staging, fmt.Sprintf("mount-%d", i+1))
		if err = s.extractTar(ctx, archive, mountDir); err != nil {
			return nil, nil, err
		}
		container.Volumes[i].BackupPath = s.archiveContentRoot(mountDir)
		if resourceRoot == archive {
			resourceRoot = ""
		}
	}
	if imageArchive != "" {
		if _, err = shell.Exec("docker load --input " + strconv.Quote(imageArchive)); err != nil {
			return nil, nil, errors.New(s.t.Get("failed to load container image: %v", err))
		}
	}
	resourceDir := ""
	if resourceRoot != "" {
		resourceDir = filepath.Join(staging, "resource")
		if err = os.MkdirAll(resourceDir, 0755); err != nil {
			return nil, nil, err
		}
		if err = s.extractTar(ctx, resourceRoot, resourceDir); err != nil {
			return nil, nil, err
		}
		_ = filepath.WalkDir(resourceDir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || entry.Name() != "meta.json" {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			var meta migrationContainerBackupMeta
			if json.Unmarshal(content, &meta) != nil || len(meta.Mounts) == 0 {
				return nil
			}
			container.Volumes = s.migrationContainerMounts(meta, filepath.Dir(path))
			return filepath.SkipAll
		})
	}
	if _, err = shell.Exec("docker image inspect " + strconv.Quote(container.Image) + " >/dev/null 2>&1"); err != nil {
		if _, pullErr := shell.Exec("docker pull " + strconv.Quote(container.Image)); pullErr != nil {
			return nil, nil, errors.New(s.t.Get("the migrated container image is unavailable: %v", pullErr))
		}
	}

	volumes, volumeCreated, warnings, err := s.restoreMigrationMounts(ctx, detail.Item.TargetName, resourceDir, container.Volumes, "", "")
	if err != nil {
		return volumeCreated, warnings, err
	}
	ports := make([]types.ContainerPort, 0, len(container.Ports))
	for _, port := range container.Ports {
		host := port.Host
		if !host.IsValid() {
			host = netip.IPv4Unspecified()
		}
		ports = append(ports, types.ContainerPort{
			ContainerStart: port.Container, ContainerEnd: port.Container, Host: host,
			HostStart: port.HostPort, HostEnd: port.HostPort, Protocol: lo.CoalesceOrEmpty(port.Protocol, "tcp"),
		})
	}
	restart := container.RestartPolicy
	if restart == "" {
		restart = "no"
	}
	create := &request.ContainerCreate{
		Name: detail.Item.TargetName, Image: container.Image, Ports: ports, Volumes: volumes,
		Network: container.Network, Labels: container.Labels, Env: container.Env,
		Entrypoint: container.Entrypoint, Command: container.Command, RestartPolicy: restart,
		Privileged: container.Privileged, AutoRemove: container.AutoRemove,
		Hostname: container.Hostname, WorkingDir: container.WorkingDir, User: container.User,
		NetworkAliases: container.NetworkAliases, StaticIP: container.StaticIP, ReadonlyRootfs: container.ReadonlyRootfs,
		OpenStdin: container.OpenStdin, PublishAllPorts: container.PublishAllPorts, Tty: container.Tty,
		CPUShares: container.CPUShares, CPUs: container.CPUs, Memory: container.Memory,
		DNS: container.DNS, ExtraHosts: container.ExtraHosts, CapAdd: container.CapAdd, CapDrop: container.CapDrop,
		Devices: container.Devices, SecurityOpt: container.SecurityOpt, Sysctls: container.Sysctls,
		Ulimits: container.Ulimits, Tmpfs: container.Tmpfs, ShmSize: container.ShmSize, Init: container.Init,
		StopSignal: container.StopSignal, StopTimeout: container.StopTimeout, Healthcheck: container.Healthcheck,
	}
	if strings.Contains(create.Network, ":") {
		return volumeCreated, warnings, errors.New(s.t.Get("the source container uses an unsupported shared network namespace: %s", create.Network))
	}
	if create.Network != "" && create.Network != "bridge" && create.Network != "host" && create.Network != "none" {
		networks, networkErr := s.containerNetworkRepo.List()
		if networkErr != nil {
			return volumeCreated, warnings, networkErr
		}
		if !slices.ContainsFunc(networks, func(network types.ContainerNetwork) bool { return network.Name == create.Network }) {
			if _, networkErr = s.containerNetworkRepo.Create(&request.ContainerNetworkCreate{
				Name: create.Network, Driver: "bridge", Ipv4: types.ContainerContainerNetwork{Enabled: true},
			}); networkErr != nil {
				return volumeCreated, warnings, errors.New(s.t.Get("failed to create target Docker network %s: %v", create.Network, networkErr))
			}
			volumeCreated = append(volumeCreated, s.t.Get("Docker network: %s", create.Network))
			warnings = append(warnings, s.t.Get("the custom Docker network was recreated with a target-assigned subnet; verify its IPAM settings"))
			if create.StaticIP != "" {
				create.StaticIP = ""
				warnings = append(warnings, s.t.Get("the source static IP was not retained because the target network uses a different subnet"))
			}
		}
	}
	s.setExternalResult(detail.Item.Key, types.MigrationItemRunning, types.MigrationStageConfiguring)
	id, err := s.containerRepo.Create(create)
	if err != nil {
		return volumeCreated, warnings, err
	}
	created := append(volumeCreated, s.t.Get("container: %s", detail.Item.TargetName))
	if !container.Running {
		_ = s.containerRepo.Stop(id)
	}
	if resourceDir == "" && len(container.Volumes) > 0 {
		warnings = append(warnings, s.t.Get("the container was recreated but its source mount data was not available from the panel backup API"))
	}
	return created, warnings, nil
}

func (s *ToolboxMigrationService) importExternalCompose(
	ctx context.Context,
	detail *types.MigrationSourceDetail,
	artifactPath string,
) ([]string, []string, error) {
	compose := detail.Compose
	if compose == nil || strings.TrimSpace(compose.Compose) == "" {
		return nil, nil, errors.New(s.t.Get("Compose configuration is missing"))
	}
	existing, err := s.composeRepo.List()
	if err != nil {
		return nil, nil, err
	}
	if slices.ContainsFunc(existing, func(item types.ContainerCompose) bool { return item.Name == detail.Item.TargetName }) {
		return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target Compose project already exists"))
	}
	targetData := filepath.Join(app.Root, "data", "migration", "containers", detail.Item.TargetName)
	sourceRoot := filepath.Dir(compose.Path)
	relocatedRoot := targetData
	if detail.Item.Type == "project" && detail.Project != nil {
		sourceRoot = detail.Project.Path
		relocatedRoot = detail.Item.TargetPath
		if relocatedRoot == "" {
			projectRoot, _ := s.settingRepo.Get(biz.SettingKeyProjectPath, filepath.Join(app.Root, "projects"))
			relocatedRoot = filepath.Join(projectRoot, detail.Item.TargetName)
		}
	}
	if entries, readErr := os.ReadDir(relocatedRoot); readErr == nil && len(entries) > 0 {
		return nil, nil, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target migration directory is not empty"))
	}
	staging, err := os.MkdirTemp(filepath.Dir(artifactPath), ".compose-import-*")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	warnings := make([]string, 0)
	created := make([]string, 0)
	payloadStaging := staging
	imageErr := s.extractTar(ctx, artifactPath, staging)
	if imageErr == nil {
		prepareAppPayload := func(root string) (string, error) {
			appArchive := ""
			_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr == nil && !entry.IsDir() && entry.Name() == "app.tar.gz" {
					appArchive = path
					return filepath.SkipAll
				}
				return nil
			})
			if appArchive == "" {
				return root, nil
			}
			appDir := filepath.Join(root, "app")
			if unpackErr := s.extractTar(ctx, appArchive, appDir); unpackErr != nil {
				return "", unpackErr
			}
			return appDir, nil
		}
		if !strings.HasSuffix(artifactPath, ".bundle.tar.gz") {
			payloadStaging, imageErr = prepareAppPayload(staging)
		} else {
			payload := ""
			codePayload := ""
			runtimePayload := ""
			originalImages := make(map[string]string, len(compose.ImageTags))
			for source := range compose.ImageTags {
				if compose.ImageSources[source] != "" {
					source = compose.ImageSources[source]
				}
				if id, inspectErr := shell.Exec("docker image inspect --format '{{.Id}}' " + strconv.Quote(source)); inspectErr == nil {
					originalImages[source] = strings.TrimSpace(id)
				}
			}
			_ = filepath.WalkDir(staging, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil || entry.IsDir() || imageErr != nil {
					return nil
				}
				if strings.HasSuffix(entry.Name(), ".tar") && s.dockerImageArchive(ctx, path) {
					if _, loadErr := shell.Exec("docker load --input " + strconv.Quote(path)); loadErr != nil {
						imageErr = loadErr
					}
				}
				if strings.HasSuffix(entry.Name(), ".tar.gz") {
					switch {
					case strings.Contains(entry.Name(), "-code-"):
						codePayload = path
					case strings.Contains(entry.Name(), "-runtime-"):
						runtimePayload = path
					default:
						payload = path
					}
				}
				return nil
			})
			if imageErr != nil {
				imageErr = fmt.Errorf("load Compose image: %w", imageErr)
			} else {
				for source, target := range compose.ImageTags {
					if compose.ImageSources[source] != "" {
						source = compose.ImageSources[source]
					}
					loadedID, inspectErr := shell.Exec("docker image inspect --format '{{.Id}}' " + strconv.Quote(target))
					if inspectErr != nil {
						loadedID, inspectErr = shell.Exec("docker image inspect --format '{{.Id}}' " + strconv.Quote(source))
						if inspectErr != nil {
							imageErr = fmt.Errorf("find loaded Compose image %s: %w", source, inspectErr)
							break
						}
						if _, tagErr := shell.Exec("docker tag " + strconv.Quote(source) + " " + strconv.Quote(target)); tagErr != nil {
							imageErr = fmt.Errorf("tag loaded Compose image %s: %w", source, tagErr)
							break
						}
					}
					if original := originalImages[source]; original != "" && original != strings.TrimSpace(loadedID) {
						if _, tagErr := shell.Exec("docker tag " + strconv.Quote(original) + " " + strconv.Quote(source)); tagErr != nil {
							imageErr = fmt.Errorf("restore target image tag %s: %w", source, tagErr)
							break
						}
					}
				}
			}
			if imageErr == nil && runtimePayload != "" {
				imageErr = s.extractTar(ctx, runtimePayload, filepath.Join(staging, "runtime"))
			}
			if imageErr == nil && codePayload != "" {
				codeDir := filepath.Join(staging, "code")
				imageErr = s.extractTar(ctx, codePayload, codeDir)
				payloadStaging = codeDir
			} else if imageErr == nil && payload != "" {
				payloadDir := filepath.Join(staging, "payload")
				if imageErr = s.extractTar(ctx, payload, payloadDir); imageErr == nil {
					payloadStaging, imageErr = prepareAppPayload(payloadDir)
				}
			}
		}
	}
	if imageErr == nil {
		hasMountMetadata := false
		_ = filepath.WalkDir(payloadStaging, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr == nil && !entry.IsDir() && (entry.Name() == "meta.json" || entry.Name() == "compose_meta.json") {
				hasMountMetadata = true
				return filepath.SkipAll
			}
			return nil
		})
		if detail.Item.Type == "project" || !hasMountMetadata {
			payload := s.archiveContentRoot(payloadStaging)
			if detail.Project != nil {
				s.removeMigrationProjectDependencies(payload, detail.Project.Type)
			}
			if copyErr := s.copyMigrationTree(ctx, payload, relocatedRoot); copyErr != nil {
				return created, warnings, copyErr
			}
			created = append(created, s.t.Get("data directory: %s", relocatedRoot))
		}
		runtimeStaging := filepath.Join(staging, "runtime")
		if detail.Item.Type == "project" {
			if info, statErr := os.Stat(runtimeStaging); statErr == nil && info.IsDir() {
				if entries, readErr := os.ReadDir(targetData); readErr == nil && len(entries) > 0 {
					return created, warnings, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("the target Runtime directory is not empty"))
				}
				if copyErr := s.copyMigrationTree(ctx, s.archiveContentRoot(runtimeStaging), targetData); copyErr != nil {
					return created, warnings, copyErr
				}
				created = append(created, s.t.Get("Runtime directory: %s", targetData))
			}
		}
		mountCreated := make([]string, 0)
		mountWarnings := make([]string, 0)
		var restoreErr error
		restoredVolumes := make(map[string]bool)
		restoredBinds := make(map[string]bool)
		roots := []string{payloadStaging}
		archives := make([]string, 0)
		_ = filepath.WalkDir(payloadStaging, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr == nil && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tar.gz") && strings.Contains(filepath.ToSlash(path), "/containers/") {
				archives = append(archives, path)
			}
			return nil
		})
		for _, archive := range archives {
			extracted, tempErr := os.MkdirTemp(filepath.Dir(payloadStaging), ".compose-container-*")
			if tempErr != nil {
				return created, warnings, tempErr
			}
			defer func(path string) { _ = os.RemoveAll(path) }(extracted)
			if tempErr = s.extractTar(ctx, archive, extracted); tempErr != nil {
				return created, warnings, tempErr
			}
			roots = append(roots, extracted)
		}
		for _, scanRoot := range roots {
			_ = filepath.WalkDir(scanRoot, func(path string, entry os.DirEntry, walkErr error) error {
				if restoreErr != nil || walkErr != nil || entry.IsDir() || entry.Name() != "meta.json" {
					return nil
				}
				content, readErr := os.ReadFile(path)
				if readErr != nil {
					return nil
				}
				var meta migrationContainerBackupMeta
				if json.Unmarshal(content, &meta) != nil || len(meta.Mounts) == 0 {
					return nil
				}
				mounts := s.migrationContainerMounts(meta, filepath.Dir(path))
				mounts = slices.DeleteFunc(mounts, func(mount types.MigrationContainerVolume) bool {
					switch mount.Type {
					case "volume":
						name := lo.CoalesceOrEmpty(mount.Name, mount.Source)
						if restoredVolumes[name] {
							return true
						}
						restoredVolumes[name] = true
					case "bind":
						if restoredBinds[mount.Source] {
							return true
						}
						restoredBinds[mount.Source] = true
					}
					return false
				})
				_, restored, restoreWarnings, mountErr := s.restoreMigrationMounts(
					ctx,
					detail.Item.TargetName,
					filepath.Dir(path),
					mounts,
					filepath.Dir(compose.Path),
					targetData,
				)
				mountCreated = append(mountCreated, restored...)
				mountWarnings = append(mountWarnings, restoreWarnings...)
				if mountErr != nil {
					restoreErr = mountErr
					return filepath.SkipAll
				}
				return nil
			})
		}
		created = append(created, mountCreated...)
		warnings = append(warnings, mountWarnings...)
		if restoreErr != nil {
			return created, warnings, restoreErr
		}
	} else if detail.Item.Type == "project" {
		return created, warnings, imageErr
	} else {
		warnings = append(warnings, s.t.Get("Compose mount data could not be read from the source backup and must be checked manually"))
	}
	composeBindRoot := targetData
	if detail.Item.Subtype == "appstore" {
		composeBindRoot = relocatedRoot
	}
	var document map[string]any
	if err = yaml.Unmarshal([]byte(compose.Compose), &document); err != nil {
		return created, warnings, fmt.Errorf("parse Compose YAML: %w", err)
	}
	services, ok := document["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return created, warnings, errors.New("compose YAML does not contain services")
	}
	relocateBind := func(source, destination string) string {
		if !filepath.IsAbs(source) {
			clean := strings.TrimPrefix(filepath.Clean(source), "."+string(os.PathSeparator))
			if clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
				return filepath.Join(composeBindRoot, clean)
			}
		}
		if sourceRoot != "" && relocatedRoot != "" {
			cleanSource := filepath.Clean(source)
			cleanRoot := filepath.Clean(sourceRoot)
			if cleanSource == cleanRoot {
				return relocatedRoot
			}
			if strings.HasPrefix(cleanSource, cleanRoot+string(os.PathSeparator)) {
				return filepath.Join(relocatedRoot, strings.TrimPrefix(cleanSource, cleanRoot+string(os.PathSeparator)))
			}
		}
		return s.migrationBindTarget(composeBindRoot, source, destination)
	}
	for _, rawService := range services {
		service, serviceOK := rawService.(map[string]any)
		if !serviceOK {
			continue
		}
		if image := cast.ToString(service["image"]); compose.ImageTags[image] != "" {
			service["image"] = compose.ImageTags[image]
		}
		volumes, volumesOK := service["volumes"].([]any)
		if !volumesOK {
			continue
		}
		for i, rawVolume := range volumes {
			switch volume := rawVolume.(type) {
			case string:
				parts := strings.Split(volume, ":")
				if len(parts) < 2 || !filepath.IsAbs(parts[0]) && !strings.HasPrefix(parts[0], ".") {
					continue
				}
				if s.isSpecialMigrationMount(parts[0]) {
					if _, statErr := os.Stat(parts[0]); statErr != nil {
						return created, warnings, fmt.Errorf("special Compose mount %s does not exist on the target", parts[0])
					}
					continue
				}
				parts[0] = relocateBind(parts[0], parts[1])
				volumes[i] = strings.Join(parts, ":")
			case map[string]any:
				if cast.ToString(volume["type"]) != "bind" {
					continue
				}
				source := lo.CoalesceOrEmpty(cast.ToString(volume["source"]), cast.ToString(volume["src"]))
				target := lo.CoalesceOrEmpty(
					cast.ToString(volume["target"]),
					cast.ToString(volume["dst"]),
					cast.ToString(volume["destination"]),
				)
				if !filepath.IsAbs(source) && !strings.HasPrefix(source, ".") {
					continue
				}
				if s.isSpecialMigrationMount(source) {
					if _, statErr := os.Stat(source); statErr != nil {
						return created, warnings, fmt.Errorf("special Compose mount %s does not exist on the target", source)
					}
					continue
				}
				volume["source"] = relocateBind(source, target)
			}
		}
		service["volumes"] = volumes
	}
	rewrittenBytes, err := yaml.Marshal(document)
	if err != nil {
		return created, warnings, err
	}
	rewritten := string(rewrittenBytes)
	s.setExternalResult(detail.Item.Key, types.MigrationItemRunning, types.MigrationStageConfiguring)
	envs := append([]types.KV(nil), compose.Envs...)
	cleanRoot := filepath.Clean(sourceRoot)
	for i := range envs {
		cleanValue := filepath.Clean(envs[i].Value)
		if cleanValue == cleanRoot {
			envs[i].Value = relocatedRoot
		} else if sourceRoot != "" && strings.HasPrefix(cleanValue, cleanRoot+string(os.PathSeparator)) {
			envs[i].Value = filepath.Join(relocatedRoot, strings.TrimPrefix(cleanValue, cleanRoot+string(os.PathSeparator)))
		}
	}
	if err = s.composeRepo.Create(detail.Item.TargetName, rewritten, envs); err != nil {
		return created, warnings, err
	}
	created = append(created, s.t.Get("Compose project: %s", detail.Item.TargetName))
	composeFile := filepath.Join(app.Root, "compose", detail.Item.TargetName, "docker-compose.yml")
	if output, configErr := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "config", "-q").CombinedOutput(); configErr != nil {
		return created, warnings, errors.New(s.t.Get("Compose validation failed: %s", strings.TrimSpace(string(output))))
	}
	if compose.Running {
		s.setExternalResult(detail.Item.Key, types.MigrationItemRunning, types.MigrationStageStarting)
		if err = s.composeRepo.Up(detail.Item.TargetName, false); err != nil {
			warnings = append(warnings, s.t.Get("Compose was restored but could not be started: %v", err))
		}
	}
	return created, warnings, nil
}

type migrationContainerBackupMeta struct {
	ContainerName string `json:"containerName"`
	Config        struct {
		Labels map[string]string `json:"Labels"`
	} `json:"config"`
	Mounts []migrationContainerBackupMount `json:"mounts"`
}

type migrationContainerBackupMount struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	RW          *bool  `json:"rw"`
	BackupPath  string `json:"backupPath"`
	Status      string `json:"status"`
}

func (s *ToolboxMigrationService) migrationContainerMounts(meta migrationContainerBackupMeta, root string) []types.MigrationContainerVolume {
	return lo.Map(meta.Mounts, func(mount migrationContainerBackupMount, _ int) types.MigrationContainerVolume {
		backupPath := mount.BackupPath
		if mount.Status != "" && mount.Status != "backed_up" {
			backupPath = ""
		}
		if backupPath != "" && !filepath.IsAbs(backupPath) {
			backupPath = filepath.Join(root, backupPath)
		}
		mode := mount.Mode
		if mode == "" && mount.RW != nil && !*mount.RW {
			mode = "ro"
		}
		return types.MigrationContainerVolume{
			Type: mount.Type, Name: mount.Name, Source: mount.Source, Destination: mount.Destination,
			Mode: mode, BackupPath: backupPath,
		}
	})
}

func (s *ToolboxMigrationService) restoreMigrationMounts(
	ctx context.Context,
	containerName string,
	resourceRoot string,
	mounts []types.MigrationContainerVolume,
	sourceRoot string,
	targetRoot string,
) ([]types.ContainerContainerVolume, []string, []string, error) {
	result := make([]types.ContainerContainerVolume, 0, len(mounts))
	created := make([]string, 0)
	warnings := make([]string, 0)
	for _, mount := range mounts {
		mode := mount.Mode
		if mode == "" {
			mode = "rw"
		}
		sourceData := mount.BackupPath
		if sourceData != "" && !filepath.IsAbs(sourceData) {
			sourceData = filepath.Join(resourceRoot, sourceData)
		}
		switch mount.Type {
		case "volume":
			name := lo.CoalesceOrEmpty(mount.Name, mount.Source)
			if name == "" {
				return nil, created, warnings, errors.New(s.t.Get("a named volume is missing its name"))
			}
			if _, err := shell.Exec("docker volume inspect " + strconv.Quote(name) + " >/dev/null 2>&1"); err == nil {
				return nil, created, warnings, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("target Docker volume %s already exists", name))
			}
			if _, err := shell.Exec("docker volume create " + strconv.Quote(name)); err != nil {
				return nil, created, warnings, err
			}
			created = append(created, s.t.Get("Docker volume: %s", name))
			if sourceData != "" {
				mountPoint, inspectErr := shell.Exec("docker volume inspect --format '{{.Mountpoint}}' " + strconv.Quote(name))
				if inspectErr != nil {
					return nil, created, warnings, inspectErr
				}
				if err := s.copyMigrationTree(ctx, sourceData, strings.TrimSpace(mountPoint)); err != nil {
					return nil, created, warnings, err
				}
			} else {
				warnings = append(warnings, s.t.Get("Docker volume %s was created empty because the source backup did not contain its data", name))
			}
			result = append(result, types.ContainerContainerVolume{Host: name, Container: mount.Destination, Mode: mode})
		case "bind":
			if s.isSpecialMigrationMount(mount.Source) {
				if _, err := os.Stat(mount.Source); err != nil {
					return nil, created, warnings, errors.New(s.t.Get("special mount %s does not exist on the target server", mount.Source))
				}
				result = append(result, types.ContainerContainerVolume{Host: mount.Source, Container: mount.Destination, Mode: mode})
				continue
			}
			target := ""
			if sourceRoot != "" && targetRoot != "" {
				cleanSource := filepath.Clean(mount.Source)
				cleanRoot := filepath.Clean(sourceRoot)
				if cleanSource == cleanRoot {
					target = targetRoot
				} else if strings.HasPrefix(cleanSource, cleanRoot+string(os.PathSeparator)) {
					target = filepath.Join(targetRoot, strings.TrimPrefix(cleanSource, cleanRoot+string(os.PathSeparator)))
				}
			}
			if target == "" {
				target = s.migrationBindTarget(filepath.Join(app.Root, "data", "migration", "containers", containerName), mount.Source, mount.Destination)
			}
			if entries, readErr := os.ReadDir(target); readErr == nil && len(entries) > 0 {
				return nil, created, warnings, fmt.Errorf("%w: %s", errMigrationConflict, s.t.Get("target bind mount directory %s is not empty", target))
			}
			if err := os.MkdirAll(target, 0755); err != nil {
				return nil, created, warnings, err
			}
			if sourceData != "" {
				if err := s.copyMigrationTree(ctx, sourceData, target); err != nil {
					return nil, created, warnings, err
				}
			} else {
				warnings = append(warnings, s.t.Get("bind mount %s was created empty because the source backup did not contain its data", mount.Destination))
			}
			result = append(result, types.ContainerContainerVolume{Host: target, Container: mount.Destination, Mode: mode})
		}
	}
	return result, created, warnings, nil
}

func (s *ToolboxMigrationService) extractTar(ctx context.Context, archive, target string) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	output, err := exec.CommandContext(ctx, "tar", "-xf", archive, "-C", target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("extract archive: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *ToolboxMigrationService) dockerImageArchive(ctx context.Context, archive string) bool {
	output, err := exec.CommandContext(ctx, "tar", "-tf", archive).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "manifest.json") || strings.Contains(string(output), "repositories")
}

func (s *ToolboxMigrationService) archiveContentRoot(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		return root
	}
	return filepath.Join(root, entries[0].Name())
}

func (s *ToolboxMigrationService) copyMigrationTree(ctx context.Context, source, target string) error {
	if source == "" {
		return errors.New("migration source directory is empty")
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	output, err := exec.CommandContext(ctx, "cp", "-a", filepath.Join(source, "."), target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("copy migration data: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *ToolboxMigrationService) migrationBindTarget(root, source, destination string) string {
	name := strings.Trim(filepath.ToSlash(filepath.Clean(source)), "/")
	if name == "." || name == "" {
		name = strings.Trim(filepath.ToSlash(filepath.Clean(destination)), "/")
	}
	var safe strings.Builder
	for _, char := range name {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.' {
			safe.WriteRune(char)
		} else if char == '/' {
			safe.WriteByte('-')
		} else {
			safe.WriteByte('_')
		}
	}
	if safe.Len() == 0 {
		safe.WriteString("data")
	}
	return filepath.Join(root, safe.String())
}

func (s *ToolboxMigrationService) isSpecialMigrationMount(path string) bool {
	clean := filepath.Clean(path)
	return clean == "/var/run/docker.sock" || clean == "/run/docker.sock" || clean == "/dev" || clean == "/proc" || clean == "/sys" ||
		strings.HasPrefix(clean, "/dev/") || strings.HasPrefix(clean, "/proc/") || strings.HasPrefix(clean, "/sys/")
}
