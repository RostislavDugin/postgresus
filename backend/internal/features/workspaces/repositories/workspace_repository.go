package workspaces_repositories

import (
	"time"

	workspaces_models "databasus-backend/internal/features/workspaces/models"
	"databasus-backend/internal/storage"

	"github.com/google/uuid"
)

type WorkspaceRepository struct{}

func (r *WorkspaceRepository) CreateWorkspace(workspace *workspaces_models.Workspace) error {
	if workspace.ID == uuid.Nil {
		workspace.ID = uuid.New()
	}

	if workspace.CreatedAt.IsZero() {
		workspace.CreatedAt = time.Now().UTC()
	}

	return storage.GetDb().Create(workspace).Error
}

func (r *WorkspaceRepository) GetWorkspaceByID(
	workspaceID uuid.UUID,
) (*workspaces_models.Workspace, error) {
	var workspace workspaces_models.Workspace

	if err := storage.GetDb().Where("id = ?", workspaceID).First(&workspace).Error; err != nil {
		return nil, err
	}

	return &workspace, nil
}

func (r *WorkspaceRepository) UpdateWorkspace(workspace *workspaces_models.Workspace) error {
	return storage.GetDb().Save(workspace).Error
}

func (r *WorkspaceRepository) DeleteWorkspace(workspaceID uuid.UUID) error {
	return storage.GetDb().Delete(&workspaces_models.Workspace{}, workspaceID).Error
}

func (r *WorkspaceRepository) GetAllWorkspaces() ([]*workspaces_models.Workspace, error) {
	var workspaces []*workspaces_models.Workspace

	err := storage.GetDb().Order("created_at DESC").Find(&workspaces).Error

	return workspaces, err
}
