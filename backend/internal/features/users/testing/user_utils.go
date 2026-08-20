package users_testing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
)

func CreateTestUser(ctx context.Context, role users_enums.UserRole) *users_dto.SignInResponseDTO {
	userID := uuid.New()
	email := fmt.Sprintf("%s-%s@test.com", strings.ToLower(string(role)), userID.String()[:8])

	hashedPassword := "$2a$10$test"
	user := &users_models.User{
		ID:                   userID,
		Email:                email,
		Name:                 "Test User",
		HashedPassword:       &hashedPassword,
		PasswordCreationTime: time.Now().UTC(),
		CreatedAt:            time.Now().UTC(),
		Role:                 role,
		Status:               users_enums.UserStatusActive,
	}

	userRepository := &users_repositories.UserRepository{}
	err := userRepository.CreateUser(user)
	if err != nil {
		panic(err)
	}

	response, err := users_services.GetUserService().GenerateAccessToken(ctx, user)
	if err != nil {
		panic(err)
	}

	response.Email = user.Email

	return response
}

func RecreateInitAdminAndGetAccess(ctx context.Context) *users_dto.SignInResponseDTO {
	RecreateInitialAdmin(ctx)

	userRepository := &users_repositories.UserRepository{}
	user, err := userRepository.GetUserByEmail(ctx, "admin")
	if err != nil {
		panic(err)
	}

	response, err := users_services.GetUserService().GenerateAccessToken(ctx, user)
	if err != nil {
		panic(err)
	}

	return response
}

func RecreateInitialAdmin(ctx context.Context) {
	userRepository := &users_repositories.UserRepository{}
	err := userRepository.RenameUserEmailForTests("admin", "admin-"+uuid.New().String())
	if err != nil {
		panic(err)
	}

	userService := users_services.GetUserService()
	err = userService.CreateInitialAdmin(ctx)
	if err != nil {
		panic(err)
	}
}
