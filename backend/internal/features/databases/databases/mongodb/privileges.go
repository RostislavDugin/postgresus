package mongodb

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"databasus-backend/internal/util/namelist"
)

const findAction = "find"

// MongoDB's Unauthorized error code, as returned by listCollections for a role without it.
const unauthorizedErrorCode = 13

// These are defined on admin and read every database, so they cover the dump whatever it targets.
var clusterWideBackupRoles = map[string]bool{
	"backup":               true,
	"root":                 true,
	"readAnyDatabase":      true,
	"readWriteAnyDatabase": true,
	"clusterAdmin":         true,
	"__system":             true,
}

// These read one database, so they only count when they are scoped to the one being dumped.
var databaseScopedBackupRoles = map[string]bool{
	"read":      true,
	"readWrite": true,
	"dbOwner":   true,
}

type backupPrivilegeScope struct {
	Username           string
	Database           string
	AuthDatabase       string
	ExcludeCollections []string
}

func (s backupPrivilegeScope) getAuthDatabaseName() string {
	if s.AuthDatabase == "" {
		return "admin"
	}

	return s.AuthDatabase
}

func checkDumpReadPrivileges(
	ctx context.Context,
	client *mongo.Client,
	scope backupPrivilegeScope,
) error {
	authDatabaseName := scope.getAuthDatabaseName()

	var userInfo bson.M
	err := client.Database(authDatabaseName).RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: scope.Username},
			{Key: "db", Value: authDatabaseName},
		}},
		{Key: "showPrivileges", Value: true},
	}).Decode(&userInfo)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	users, isUserList := userInfo["users"].(bson.A)
	if !isUserList || len(users) == 0 {
		return errors.New("insufficient permissions for backup. User not found")
	}

	user, isUserDocument := users[0].(bson.M)
	if !isUserDocument {
		return errors.New("insufficient permissions for backup. Could not parse user info")
	}

	account := grantedAccount{Scope: scope, UsersInfo: user}

	if account.hasBackupRole() {
		return nil
	}

	if account.isFindGrantedOnWholeDatabase() {
		return nil
	}

	return account.checkPerCollectionFindPrivileges(ctx, client)
}

type grantedAccount struct {
	Scope     backupPrivilegeScope
	UsersInfo bson.M
}

func (a grantedAccount) getGrantedRoleNames() []string {
	var grantedRoleNames []string

	for _, role := range a.getRoles() {
		if roleName, _ := role["role"].(string); roleName != "" {
			grantedRoleNames = append(grantedRoleNames, roleName)
		}
	}

	return grantedRoleNames
}

func (a grantedAccount) hasBackupRole() bool {
	for _, role := range a.getRoles() {
		roleName, _ := role["role"].(string)
		roleDatabaseName, _ := role["db"].(string)

		// The all-database built-ins exist only on admin; a custom role that merely shares
		// their name on another database grants none of their reach.
		if clusterWideBackupRoles[roleName] && roleDatabaseName == "admin" {
			return true
		}

		if databaseScopedBackupRoles[roleName] &&
			(roleDatabaseName == a.Scope.Database || roleDatabaseName == "") {
			return true
		}
	}

	return false
}

func (a grantedAccount) getRoles() []bson.M {
	roles, isRoleList := a.UsersInfo["roles"].(bson.A)
	if !isRoleList {
		return nil
	}

	roleDocuments := make([]bson.M, 0, len(roles))

	for _, roleEntry := range roles {
		if role, isRoleDocument := roleEntry.(bson.M); isRoleDocument {
			roleDocuments = append(roleDocuments, role)
		}
	}

	return roleDocuments
}

// A find privilege counts for the whole dump only when its resource is the database itself, the
// cluster, or any resource. A privilege naming one collection covers just that collection, which
// is what the per-collection pass below has to work out.
func (a grantedAccount) isFindGrantedOnWholeDatabase() bool {
	for _, privilege := range a.getInheritedPrivileges() {
		resource, isResourceDocument := privilege["resource"].(bson.M)
		if !isResourceDocument || !hasFindAction(privilege) {
			continue
		}

		if isCluster, _ := resource["cluster"].(bool); isCluster {
			return true
		}

		resourceCollectionName, _ := resource["collection"].(string)
		if resourceCollectionName != "" {
			continue
		}

		resourceDatabaseName, _ := resource["db"].(string)
		if resourceDatabaseName == a.Scope.Database || resourceDatabaseName == "" {
			return true
		}
	}

	return false
}

func (a grantedAccount) checkPerCollectionFindPrivileges(
	ctx context.Context,
	client *mongo.Client,
) error {
	dumpedCollectionNames, err := readDumpedCollectionNames(ctx, client, a.Scope)
	if err != nil {
		// A role that cannot list the collections certainly cannot dump them, but a transport
		// failure must keep saying what it was rather than pose as a permission problem.
		if !isUnauthorizedError(err) {
			return err
		}

		return a.newInsufficientPrivilegesError(nil)
	}

	readableCollectionNames := a.getCollectionsGrantingFind()

	var unreadableCollectionNames []string

	for _, collectionName := range dumpedCollectionNames {
		if !slices.Contains(readableCollectionNames, collectionName) {
			unreadableCollectionNames = append(unreadableCollectionNames, collectionName)
		}
	}

	if len(unreadableCollectionNames) == 0 {
		return nil
	}

	return a.newInsufficientPrivilegesError(unreadableCollectionNames)
}

func (a grantedAccount) getCollectionsGrantingFind() []string {
	var readableCollectionNames []string

	for _, privilege := range a.getInheritedPrivileges() {
		resource, isResourceDocument := privilege["resource"].(bson.M)
		if !isResourceDocument || !hasFindAction(privilege) {
			continue
		}

		resourceCollectionName, _ := resource["collection"].(string)
		if resourceCollectionName == "" {
			continue
		}

		resourceDatabaseName, _ := resource["db"].(string)
		if resourceDatabaseName == a.Scope.Database || resourceDatabaseName == "" {
			readableCollectionNames = append(readableCollectionNames, resourceCollectionName)
		}
	}

	return readableCollectionNames
}

func (a grantedAccount) getInheritedPrivileges() []bson.M {
	inheritedPrivileges, isPrivilegeList := a.UsersInfo["inheritedPrivileges"].(bson.A)
	if !isPrivilegeList {
		return nil
	}

	privileges := make([]bson.M, 0, len(inheritedPrivileges))

	for _, privilegeEntry := range inheritedPrivileges {
		if privilege, isPrivilegeDocument := privilegeEntry.(bson.M); isPrivilegeDocument {
			privileges = append(privileges, privilege)
		}
	}

	return privileges
}

func hasFindAction(privilege bson.M) bool {
	actions, isActionList := privilege["actions"].(bson.A)
	if !isActionList {
		return false
	}

	for _, actionEntry := range actions {
		if action, isActionName := actionEntry.(string); isActionName && action == findAction {
			return true
		}
	}

	return false
}

func isUnauthorizedError(err error) bool {
	var commandError mongo.CommandError
	if errors.As(err, &commandError) {
		return commandError.Code == unauthorizedErrorCode
	}

	return false
}

func readDumpedCollectionNames(
	ctx context.Context,
	client *mongo.Client,
	scope backupPrivilegeScope,
) ([]string, error) {
	collectionNames, err := client.Database(scope.Database).
		ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	excludedCollectionNames := namelist.NormalizeUniqueNames(scope.ExcludeCollections)
	dumpedCollectionNames := make([]string, 0, len(collectionNames))

	for _, collectionName := range collectionNames {
		if slices.Contains(excludedCollectionNames, collectionName) {
			continue
		}

		dumpedCollectionNames = append(dumpedCollectionNames, collectionName)
	}

	return dumpedCollectionNames, nil
}

func (a grantedAccount) newInsufficientPrivilegesError(
	unreadableCollectionNames []string,
) error {
	if len(unreadableCollectionNames) > 0 {
		return fmt.Errorf(
			"insufficient permissions for backup. Cannot read collections: %s. "+
				"Required: 'read' role on database '%s' OR 'backup' role on admin OR 'readAnyDatabase' role",
			namelist.FormatTruncatedNames(unreadableCollectionNames),
			a.Scope.Database,
		)
	}

	return fmt.Errorf(
		"insufficient permissions for backup. Current roles: %s. "+
			"Required: 'read' role on database '%s' OR 'backup' role on admin OR 'readAnyDatabase' role",
		strings.Join(a.getGrantedRoleNames(), ", "),
		a.Scope.Database,
	)
}
