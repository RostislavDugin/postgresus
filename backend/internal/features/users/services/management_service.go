package users_services

import (
	"errors"
	"fmt"
	"time"

	user_enums "databasus-backend/internal/features/users/enums"
	user_interfaces "databasus-backend/internal/features/users/interfaces"
	user_models "databasus-backend/internal/features/users/models"
	user_repositories "databasus-backend/internal/features/users/repositories"

	"github.com/google/uuid"
)

type UserManagementService struct {
	userRepository *user_repositories.UserRepository
	auditLogWriter user_interfaces.AuditLogWriter
}

func (s *UserManagementService) SetAuditLogWriter(writer user_interfaces.AuditLogWriter) {
	s.auditLogWriter = writer
}

func (s *UserManagementService) GetUsers(
	currentUser *user_models.User,
	limit, offset int,
	beforeCreatedAt *time.Time,
	query string,
) ([]*user_models.User, int64, error) {
	if !currentUser.CanManageUsers() {
		return nil, 0, errors.New("insufficient permissions to list users")
	}

	return s.userRepository.GetUsers(limit, offset, beforeCreatedAt, query)
}

func (s *UserManagementService) GetUserProfile(
	userID uuid.UUID,
	requestedBy *user_models.User,
) (*user_models.User, error) {
	// Users can view their own profile, admins can view any profile
	if userID != requestedBy.ID && !requestedBy.CanManageUsers() {
		return nil, errors.New("insufficient permissions to view user profile")
	}

	return s.userRepository.GetUserByID(userID)
}

func (s *UserManagementService) DeactivateUser(
	userID uuid.UUID,
	deactivatedBy *user_models.User,
) error {
	if !deactivatedBy.CanManageUsers() {
		return errors.New("insufficient permissions to deactivate users")
	}

	// Don't allow deactivating self
	if userID == deactivatedBy.ID {
		return errors.New("cannot deactivate your own account")
	}

	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Only user with email "admin" can deactivate ADMIN users
	if user.Role == user_enums.UserRoleAdmin && deactivatedBy.Email != "admin" {
		return errors.New("only the root admin user can deactivate admin accounts")
	}

	if err := s.userRepository.UpdateUserStatus(userID, user_enums.UserStatusInactive); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(
			fmt.Sprintf("User deactivated: %s", user.Email),
			&deactivatedBy.ID,
			nil,
		)
	}

	return nil
}

func (s *UserManagementService) ActivateUser(
	userID uuid.UUID,
	activatedBy *user_models.User,
) error {
	if !activatedBy.CanManageUsers() {
		return errors.New("insufficient permissions to activate users")
	}

	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Only user with email "admin" can activate ADMIN users
	if user.Role == user_enums.UserRoleAdmin && activatedBy.Email != "admin" {
		return errors.New("only the root admin user can activate admin accounts")
	}

	if err := s.userRepository.UpdateUserStatus(userID, user_enums.UserStatusActive); err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(
			fmt.Sprintf("User activated: %s", user.Email),
			&activatedBy.ID,
			nil,
		)
	}

	return nil
}

func (s *UserManagementService) ChangeUserRole(
	userID uuid.UUID,
	newRole user_enums.UserRole,
	changedBy *user_models.User,
) error {
	if !changedBy.CanManageUsers() {
		return errors.New("insufficient permissions to change user roles")
	}

	// Validate role
	if !newRole.IsValid() {
		return errors.New("invalid user role")
	}

	// Don't allow changing own role
	if userID == changedBy.ID {
		return errors.New("cannot change your own role")
	}

	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Only user with email "admin" can promote users to ADMIN or demote ADMIN users
	if (newRole == user_enums.UserRoleAdmin || user.Role == user_enums.UserRoleAdmin) &&
		changedBy.Email != "admin" {
		return errors.New(
			"only the root admin user can promote users to admin or demote admin users",
		)
	}

	if err := s.userRepository.UpdateUserRole(userID, newRole); err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(
			fmt.Sprintf("User role changed: %s from %s to %s", user.Email, user.Role, newRole),
			&changedBy.ID,
			nil,
		)
	}

	return nil
}
