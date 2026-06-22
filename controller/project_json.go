package controller

import (
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/usecase"
)

type projectPlanJSON struct {
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	MembershipType  string `json:"membership_type"`
	Price           int    `json:"price"`
	MaxSupporters   *int   `json:"max_supporters"`
	Benefits        string `json:"benefits"`
	IsActive        bool   `json:"is_active"`
	SupporterCount  int    `json:"supporter_count"`
}

type projectImageJSON struct {
	ID       string `json:"id"`
	ImageURL string `json:"image_url"`
	IsMain   bool   `json:"is_main"`
	Order    int    `json:"order"`
}

type projectSummaryJSON struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	TargetAmount    int64              `json:"target_amount"`
	CurrentAmount   int64              `json:"current_amount"`
	SupporterCount  int                `json:"supporter_count"`
	DurationType    string             `json:"duration_type"`
	StartDate       string             `json:"start_date"`
	EndDate         *string            `json:"end_date"`
	Status          string             `json:"status"`
	ImageURL        *string            `json:"image_url"`
	Plans           []projectPlanJSON  `json:"plans"`
	Seller          *userJSON          `json:"seller,omitempty"`
}

type listProjectsResponse struct {
	Projects []projectSummaryJSON `json:"projects"`
	Total    int64                `json:"total"`
	Limit    int                  `json:"limit"`
	Offset   int                  `json:"offset"`
}

type projectDetailResponse struct {
	projectSummaryJSON
	Images        []projectImageJSON `json:"images"`
	DeleteRequest bool               `json:"delete_request"`
}

func listProjectsJSON(rows []usecase.ProjectView, total int64, limit, offset int) listProjectsResponse {
	items := make([]projectSummaryJSON, 0, len(rows))
	for i := range rows {
		items = append(items, projectSummaryToJSON(&rows[i]))
	}
	if limit <= 0 {
		limit = 20
	}
	return listProjectsResponse{
		Projects: items,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}
}

func projectDetailJSON(v *usecase.ProjectView) projectDetailResponse {
	return projectDetailResponse{
		projectSummaryJSON: projectSummaryToJSON(v),
		Images:             projectImagesToJSON(v.Images),
		DeleteRequest:      v.Project.DeleteRequested,
	}
}

func projectSummaryToJSON(v *usecase.ProjectView) projectSummaryJSON {
	p := v.Project
	out := projectSummaryJSON{
		ID:             p.ID.String(),
		Title:          p.Title,
		Description:    p.Description,
		TargetAmount:   p.TargetAmount,
		CurrentAmount:  p.CurrentAmount,
		SupporterCount: p.SupporterCount,
		DurationType:   string(p.DurationType),
		StartDate:      p.StartDate.UTC().Format(time.RFC3339),
		Status:         string(p.Status),
		ImageURL:       p.MainImageURL,
		Plans:          projectPlansToJSON(v.Plans),
	}
	if p.EndDate != nil {
		s := p.EndDate.UTC().Format(time.RFC3339)
		out.EndDate = &s
	}
	if v.Seller != nil {
		seller := userToJSON(v.Seller)
		out.Seller = &seller
	}
	return out
}

func projectPlansToJSON(plans []entity.ProjectPlan) []projectPlanJSON {
	out := make([]projectPlanJSON, 0, len(plans))
	for i := range plans {
		pl := &plans[i]
		out = append(out, projectPlanJSON{
			ID:             pl.ID.String(),
			ProjectID:      pl.ProjectID.String(),
			MembershipType: string(pl.MembershipType),
			Price:          pl.Price,
			MaxSupporters:  pl.MaxSupporters,
			Benefits:       pl.Benefits,
			IsActive:       pl.IsActive,
			SupporterCount: pl.SupporterCount,
		})
	}
	return out
}

func projectImagesToJSON(images []entity.ProjectImage) []projectImageJSON {
	out := make([]projectImageJSON, 0, len(images))
	for i := range images {
		img := &images[i]
		out = append(out, projectImageJSON{
			ID:       img.ID.String(),
			ImageURL: img.ImageURL,
			IsMain:   img.IsMain,
			Order:    img.SortOrder,
		})
	}
	return out
}
