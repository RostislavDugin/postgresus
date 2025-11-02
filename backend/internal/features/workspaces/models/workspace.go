package workspaces_models

import (
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID        uuid.UUID `json:"id"        gorm:"column:id"`
	Name      string    `json:"name"      gorm:"column:name"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (Workspace) TableName() string {
	return "workspaces"
}

func (p *Workspace) UpdateFromDTO(updateDTO *Workspace) {
	p.Name = updateDTO.Name
}
