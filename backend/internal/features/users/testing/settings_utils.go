package users_testing

import (
	"context"

	users_repositories "databasus-backend/internal/features/users/repositories"
)

func EnableMemberInvitations(ctx context.Context) {
	updateUsersSetting(ctx, "is_allow_member_invitations", true)
}

func DisableMemberInvitations(ctx context.Context) {
	updateUsersSetting(ctx, "is_allow_member_invitations", false)
}

func EnableExternalRegistrations(ctx context.Context) {
	updateUsersSetting(ctx, "is_allow_external_registrations", true)
}

func DisableExternalRegistrations(ctx context.Context) {
	updateUsersSetting(ctx, "is_allow_external_registrations", false)
}

func EnableMemberWorkspaceCreation(ctx context.Context) {
	updateUsersSetting(ctx, "is_member_allowed_to_create_workspaces", true)
}

func DisableMemberWorkspaceCreation(ctx context.Context) {
	updateUsersSetting(ctx, "is_member_allowed_to_create_workspaces", false)
}

func ResetSettingsToDefaults(ctx context.Context) {
	repository := &users_repositories.UsersSettingsRepository{}
	settings, err := repository.GetSettings(ctx)
	if err != nil {
		panic(err)
	}

	settings.IsAllowExternalRegistrations = true
	settings.IsAllowMemberInvitations = true
	settings.IsMemberAllowedToCreateWorkspaces = true

	err = repository.UpdateSettings(ctx, settings)
	if err != nil {
		panic(err)
	}
}

func updateUsersSetting(ctx context.Context, column string, value bool) {
	repository := &users_repositories.UsersSettingsRepository{}
	settings, err := repository.GetSettings(ctx)
	if err != nil {
		panic(err)
	}

	switch column {
	case "is_allow_member_invitations":
		settings.IsAllowMemberInvitations = value
	case "is_allow_external_registrations":
		settings.IsAllowExternalRegistrations = value
	case "is_member_allowed_to_create_workspaces":
		settings.IsMemberAllowedToCreateWorkspaces = value
	}

	err = repository.UpdateSettings(ctx, settings)
	if err != nil {
		panic(err)
	}
}
