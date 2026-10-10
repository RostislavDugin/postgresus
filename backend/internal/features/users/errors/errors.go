package users_errors

import (
	"errors"
	"fmt"

	users_models "databasus-backend/internal/features/users/models"
)

var ErrInsufficientPermissionsToInviteUsers = errors.New("insufficient permissions to invite users")

var ErrPasswordTooShort = fmt.Errorf(
	"the password must be at least %d characters long",
	users_models.MinPasswordLength,
)

var (
	ErrConsoleAccountEmailInvalid = errors.New("the email is not a valid email address")
	ErrConsoleAccountRoleInvalid  = errors.New("the role must be admin or member")
	// The first account administers the instance, so a member there would leave
	// nobody able to grant the administrator role.
	ErrConsoleAccountFirstMustBeAdmin = errors.New(
		"this instance has no account yet, and its first account administers it: create it with the admin role",
	)
	ErrConsoleAccountExistsInAnotherState = errors.New(
		"an account with this email already exists with another role, is deactivated, " +
			"has not completed sign-up or has no password; nothing changed",
	)
)

var (
	ErrEmailNotConfigured = errors.New("the instance has no mail server configured")
	ErrAdminEmailMissing  = errors.New("your account has no valid email address to send a test email to")
)

var (
	// The instance could not deliver the code, so it admits nobody through
	// password sign-in while the second factor is on.
	ErrSignInCodeNotSent = errors.New(
		"could not send the sign-in code, please try again later",
	)
	ErrTooManySignInCodes = errors.New(
		"too many codes requested for this account, please try again later",
	)
	ErrSignInCodeResentTooSoon = errors.New(
		"a code was sent less than a minute ago, please wait before requesting another",
	)
	ErrPendingSignInNotUsable = errors.New(
		"this sign-in can no longer be completed, please sign in again",
	)
	ErrSignInCodeIncorrect = errors.New("the code is incorrect")
)
