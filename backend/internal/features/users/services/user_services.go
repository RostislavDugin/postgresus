package users_services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"postgresus-backend/internal/config"
	users_dto "postgresus-backend/internal/features/users/dto"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_interfaces "postgresus-backend/internal/features/users/interfaces"
	users_models "postgresus-backend/internal/features/users/models"
	users_repositories "postgresus-backend/internal/features/users/repositories"
)

type UserService struct {
	userRepository      *users_repositories.UserRepository
	secretKeyRepository *users_repositories.SecretKeyRepository
	settingsService     *SettingsService
	auditLogWriter      users_interfaces.AuditLogWriter
}

func (s *UserService) SetAuditLogWriter(writer users_interfaces.AuditLogWriter) {
	s.auditLogWriter = writer
}

func (s *UserService) SignUp(request *users_dto.SignUpRequestDTO) error {
	existingUser, err := s.userRepository.GetUserByEmail(request.Email)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil && existingUser.Status != users_enums.UserStatusInvited {
		return errors.New("user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)

	// If user exists with INVITED status, activate them and set password
	if existingUser != nil && existingUser.Status == users_enums.UserStatusInvited {
		if err := s.userRepository.UpdateUserPassword(existingUser.ID, hashedPasswordStr); err != nil {
			return fmt.Errorf("failed to set password: %w", err)
		}

		if err := s.userRepository.UpdateUserStatus(existingUser.ID, users_enums.UserStatusActive); err != nil {
			return fmt.Errorf("failed to activate user: %w", err)
		}

		name := request.Name
		if err := s.userRepository.UpdateUserInfo(existingUser.ID, &name, nil); err != nil {
			return fmt.Errorf("failed to update name: %w", err)
		}

		s.auditLogWriter.WriteAuditLog(
			fmt.Sprintf("Invited user completed registration: %s", existingUser.Email),
			&existingUser.ID,
			nil,
		)

		return nil
	}

	// Get settings to check registration policy for new users
	settings, err := s.settingsService.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// Check if external registrations are allowed
	if !settings.IsAllowExternalRegistrations {
		return errors.New("external registration is disabled")
	}

	user := &users_models.User{
		ID:                   uuid.New(),
		Email:                request.Email,
		Name:                 request.Name,
		HashedPassword:       &hashedPasswordStr,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 users_enums.UserRoleMember,
		Status:               users_enums.UserStatusActive,
		CreatedAt:            time.Now().UTC(),
	}

	if err := s.userRepository.CreateUser(user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	s.auditLogWriter.WriteAuditLog(
		fmt.Sprintf("User registered with email: %s", user.Email),
		&user.ID,
		nil,
	)

	return nil
}

func (s *UserService) SignIn(
	request *users_dto.SignInRequestDTO,
) (*users_dto.SignInResponseDTO, error) {
	user, err := s.userRepository.GetUserByEmail(request.Email)
	if err != nil {
		return nil, errors.New("user with this email does not exist")
	}

	if user == nil {
		usersCount, err := s.userRepository.GetUsersCount()
		if err != nil {
			return nil, fmt.Errorf("failed to get users count: %w", err)
		}

		if usersCount == 1 {
			return nil, errors.New(
				"user with this email does not exist, seems you need to sign in as \"admin\"",
			)
		}

		return nil, errors.New("user with this email does not exist")
	}

	if user.Status == users_enums.UserStatusInvited {
		return nil, errors.New("user account is not passed sign up yet")
	}

	if user.Status != users_enums.UserStatusActive {
		return nil, errors.New("user account is deactivated")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.HashedPassword), []byte(request.Password))
	if err != nil {
		return nil, errors.New("password is incorrect")
	}

	response, err := s.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	s.auditLogWriter.WriteAuditLog(
		fmt.Sprintf("User signed in with email: %s", user.Email),
		&user.ID,
		nil,
	)

	return response, nil
}

func (s *UserService) GetUserFromToken(token string) (*users_models.User, error) {
	secretKey, err := s.secretKeyRepository.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		userIDStr, ok := claims["sub"].(string)
		if !ok {
			return nil, errors.New("invalid token claims")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, errors.New("invalid token claims")
		}

		user, err := s.userRepository.GetUserByID(userID)
		if err != nil {
			return nil, err
		}

		// Check if user is active
		if !user.IsActiveUser() {
			return nil, errors.New("user account is deactivated")
		}

		if passwordCreationTimeUnix, ok := claims["passwordCreationTime"].(float64); ok {
			tokenPasswordTime := time.Unix(int64(passwordCreationTimeUnix), 0)

			tokenTimeSeconds := tokenPasswordTime.Truncate(time.Second)
			userTimeSeconds := user.PasswordCreationTime.Truncate(time.Second)

			if !tokenTimeSeconds.Equal(userTimeSeconds) {
				return nil, errors.New("password has been changed, please sign in again")
			}
		} else {
			return nil, errors.New("invalid token claims: missing password creation time")
		}

		return user, nil
	}

	return nil, errors.New("invalid token")
}

func (s *UserService) GenerateAccessToken(
	user *users_models.User,
) (*users_dto.SignInResponseDTO, error) {
	secretKey, err := s.secretKeyRepository.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	tenYearsExpiration := time.Now().UTC().Add(time.Hour * 24 * 365 * 10)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                  user.ID.String(),
		"exp":                  tenYearsExpiration.Unix(),
		"iat":                  time.Now().UTC().Unix(),
		"role":                 string(user.Role),
		"passwordCreationTime": user.PasswordCreationTime.Unix(),
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &users_dto.SignInResponseDTO{
		UserID: user.ID,
		Token:  tokenString,
	}, nil
}

func (s *UserService) CreateInitialAdmin() error {
	return s.userRepository.CreateInitialAdmin()
}

func (s *UserService) IsRootAdminHasPassword() (bool, error) {
	admin, err := s.userRepository.GetUserByEmail("admin")

	if err != nil {
		return false, fmt.Errorf("failed to get admin user: %w", err)
	}

	if admin == nil {
		return false, errors.New("admin user does not exist")
	}

	return admin.HasPassword(), nil
}

func (s *UserService) SetRootAdminPassword(password string) error {
	admin, err := s.userRepository.GetUserByEmail("admin")
	if err != nil {
		return fmt.Errorf("failed to get admin user: %w", err)
	}

	if admin == nil {
		return errors.New("admin user does not exist")
	}

	if admin.HasPassword() {
		return errors.New("admin password is already set")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepository.UpdateUserPassword(admin.ID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to set admin password: %w", err)
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(
			"Admin password set",
			&admin.ID,
			nil,
		)
	}

	return nil
}

func (s *UserService) ChangeUserPasswordByEmail(email string, newPassword string) error {
	user, err := s.userRepository.GetUserByEmail(email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	return s.ChangeUserPassword(user.ID, newPassword)
}

func (s *UserService) ChangeUserPassword(userID uuid.UUID, newPassword string) error {
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if !user.HasPassword() {
		return errors.New("user has no password set")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.userRepository.UpdateUserPassword(userID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.auditLogWriter.WriteAuditLog(
		"Password changed",
		&userID,
		nil,
	)

	return nil
}

func (s *UserService) InviteUser(
	request *users_dto.InviteUserRequestDTO,
	invitedBy *users_models.User,
) (*users_dto.InviteUserResponseDTO, error) {
	// Get settings to check permissions
	settings, err := s.settingsService.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Check if user has permission to invite
	if !invitedBy.CanInviteUsers(settings) {
		return nil, errors.New("insufficient permissions to invite users")
	}

	// Check if user already exists
	existingUser, err := s.userRepository.GetUserByEmail(request.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	user := &users_models.User{
		ID:                   uuid.New(),
		Email:                request.Email,
		Name:                 "User",
		HashedPassword:       nil,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 users_enums.UserRoleMember,
		Status:               users_enums.UserStatusInvited,
		CreatedAt:            time.Now().UTC(),
	}

	if err := s.userRepository.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create invited user: %w", err)
	}

	message := fmt.Sprintf("User invited: %s", request.Email)
	if request.IntendedWorkspaceID != nil {
		message += fmt.Sprintf(" for workspace %s", request.IntendedWorkspaceID.String())
	}
	s.auditLogWriter.WriteAuditLog(
		message,
		&invitedBy.ID,
		request.IntendedWorkspaceID,
	)

	return &users_dto.InviteUserResponseDTO{
		ID:                    user.ID,
		Email:                 user.Email,
		IntendedWorkspaceID:   request.IntendedWorkspaceID,
		IntendedWorkspaceRole: request.IntendedWorkspaceRole,
		CreatedAt:             user.CreatedAt,
	}, nil
}

func (s *UserService) GetUserByID(userID uuid.UUID) (*users_models.User, error) {
	return s.userRepository.GetUserByID(userID)
}

func (s *UserService) GetUserByEmail(email string) (*users_models.User, error) {
	return s.userRepository.GetUserByEmail(email)
}

func (s *UserService) GetCurrentUserProfile(
	user *users_models.User,
) *users_dto.UserProfileResponseDTO {
	return &users_dto.UserProfileResponseDTO{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		IsActive:  user.IsActiveUser(),
		CreatedAt: user.CreatedAt,
	}
}

func (s *UserService) UpdateUserInfo(
	userID uuid.UUID,
	request *users_dto.UpdateUserInfoRequestDTO,
) error {
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user.Email == "admin" && request.Email != nil && *request.Email != user.Email {
		return errors.New("admin email cannot be changed")
	}

	if request.Email != nil && *request.Email != user.Email {
		existingUser, err := s.userRepository.GetUserByEmail(*request.Email)
		if err != nil {
			return fmt.Errorf("failed to check email: %w", err)
		}
		if existingUser != nil {
			return errors.New("email is already taken by another user")
		}
	}

	if err := s.userRepository.UpdateUserInfo(userID, request.Name, request.Email); err != nil {
		return fmt.Errorf("failed to update user info: %w", err)
	}

	s.auditLogWriter.WriteAuditLog("User info updated", &userID, nil)
	return nil
}

func (s *UserService) HandleGitHubOAuth(
	code, redirectUri string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGitHubOAuthWithEndpoint(
		code,
		redirectUri,
		github.Endpoint,
		"https://api.github.com/user",
	)
}

func (s *UserService) handleGitHubOAuthWithEndpoint(
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	env := config.GetEnv()

	oauthConfig := &oauth2.Config{
		ClientID:     env.GitHubClientID,
		ClientSecret: env.GitHubClientSecret,
		RedirectURL:  redirectUri,
		Endpoint:     endpoint,
		Scopes:       []string{"user:email"},
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get(userAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var githubUser struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Login string `json:"login"`
	}

	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	email := githubUser.Email
	if email == "" {
		email, err = s.fetchGitHubPrimaryEmail(client, userAPIURL)
		if err != nil {
			return nil, err
		}
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	oauthID := fmt.Sprintf("%d", githubUser.ID)
	return s.getOrCreateUserFromOAuth(oauthID, email, name, "github")
}

func (s *UserService) HandleGoogleOAuth(
	code, redirectUri string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGoogleOAuthWithEndpoint(
		code,
		redirectUri,
		google.Endpoint,
		"https://www.googleapis.com/oauth2/v2/userinfo",
	)
}

func (s *UserService) handleGoogleOAuthWithEndpoint(
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	env := config.GetEnv()

	oauthConfig := &oauth2.Config{
		ClientID:     env.GoogleClientID,
		ClientSecret: env.GoogleClientSecret,
		RedirectURL:  redirectUri,
		Endpoint:     endpoint,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get(userAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	if googleUser.Email == "" {
		return nil, errors.New("google account has no email")
	}

	name := googleUser.Name
	if name == "" {
		name = "User"
	}

	return s.getOrCreateUserFromOAuth(googleUser.ID, googleUser.Email, name, "google")
}

func (s *UserService) getOrCreateUserFromOAuth(
	oauthID, email, name, provider string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	var existingUser *users_models.User
	var err error

	if provider == "github" {
		existingUser, err = s.userRepository.GetUserByGitHubOAuthID(oauthID)
	} else {
		existingUser, err = s.userRepository.GetUserByGoogleOAuthID(oauthID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check OAuth ID: %w", err)
	}

	if existingUser != nil {
		tokenResponse, err := s.GenerateAccessToken(existingUser)
		if err != nil {
			return nil, err
		}

		if s.auditLogWriter != nil {
			s.auditLogWriter.WriteAuditLog(
				fmt.Sprintf("User signed in via %s", provider),
				&existingUser.ID,
				nil,
			)
		}

		return &users_dto.OAuthCallbackResponseDTO{
			UserID:    tokenResponse.UserID,
			Email:     existingUser.Email,
			Token:     tokenResponse.Token,
			IsNewUser: false,
		}, nil
	}

	userByEmail, err := s.userRepository.GetUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	if userByEmail != nil {
		if userByEmail.Status == users_enums.UserStatusInvited {
			if err := s.userRepository.UpdateUserStatus(userByEmail.ID, users_enums.UserStatusActive); err != nil {
				return nil, fmt.Errorf("failed to activate user: %w", err)
			}

			if err := s.userRepository.UpdateUserInfo(userByEmail.ID, &name, nil); err != nil {
				return nil, fmt.Errorf("failed to update name: %w", err)
			}
		}

		oauthColumn := "github_oauth_id"
		if provider == "google" {
			oauthColumn = "google_oauth_id"
		}

		if err := s.userRepository.LinkOAuthID(userByEmail.ID, oauthColumn, oauthID); err != nil {
			return nil, fmt.Errorf("failed to link OAuth ID: %w", err)
		}

		user, err := s.userRepository.GetUserByID(userByEmail.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated user: %w", err)
		}

		tokenResponse, err := s.GenerateAccessToken(user)
		if err != nil {
			return nil, err
		}

		if s.auditLogWriter != nil {
			s.auditLogWriter.WriteAuditLog(
				fmt.Sprintf("%s OAuth linked to existing account", provider),
				&user.ID,
				nil,
			)
		}

		return &users_dto.OAuthCallbackResponseDTO{
			UserID:    tokenResponse.UserID,
			Email:     user.Email,
			Token:     tokenResponse.Token,
			IsNewUser: false,
		}, nil
	}

	settings, err := s.settingsService.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	if !settings.IsAllowExternalRegistrations {
		return nil, errors.New("external registration is disabled")
	}

	var githubOAuthID *string
	var googleOAuthID *string
	if provider == "github" {
		githubOAuthID = &oauthID
	} else {
		googleOAuthID = &oauthID
	}

	newUser := &users_models.User{
		ID:                   uuid.New(),
		Email:                email,
		Name:                 name,
		HashedPassword:       nil,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 users_enums.UserRoleMember,
		Status:               users_enums.UserStatusActive,
		GitHubOAuthID:        githubOAuthID,
		GoogleOAuthID:        googleOAuthID,
		CreatedAt:            time.Now().UTC(),
	}

	if err := s.userRepository.CreateUser(newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	tokenResponse, err := s.GenerateAccessToken(newUser)
	if err != nil {
		return nil, err
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(
			fmt.Sprintf("User registered via %s OAuth: %s", provider, email),
			&newUser.ID,
			nil,
		)
	}

	return &users_dto.OAuthCallbackResponseDTO{
		UserID:    tokenResponse.UserID,
		Email:     newUser.Email,
		Token:     tokenResponse.Token,
		IsNewUser: true,
	}, nil
}

func (s *UserService) fetchGitHubPrimaryEmail(
	client *http.Client,
	userAPIURL string,
) (string, error) {
	emailsURL := "https://api.github.com/user/emails"
	if userAPIURL != "https://api.github.com/user" {
		baseURL := userAPIURL[:len(userAPIURL)-len("/user")]
		emailsURL = baseURL + "/user/emails"
	}

	resp, err := client.Get(emailsURL)
	if err != nil {
		return "", fmt.Errorf("failed to get user emails: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("github account has no accessible email")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read emails response: %w", err)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("failed to parse emails: %w", err)
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", errors.New("github account has no accessible email")
}
