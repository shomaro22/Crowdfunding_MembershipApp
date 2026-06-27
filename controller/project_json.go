package controller

import (
	"time"

	"crowdfund-app/usecase"
)

type projectPlanJSON struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	MembershipType string `json:"membership_type"`
	Price          int    `json:"price"`
	MaxSupporters  *int   `json:"max_supporters"`
	Benefits       string `json:"benefits"`
	IsActive       bool   `json:"is_active"`
	SupporterCount int    `json:"supporter_count"`
}

type projectImageJSON struct {
	ID       string `json:"id"`
	ImageURL string `json:"image_url"`
	IsMain   bool   `json:"is_main"`
	Order    int    `json:"order"`
}

type projectSummaryJSON struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	TargetAmount   int64             `json:"target_amount"`
	CurrentAmount  int64             `json:"current_amount"`
	SupporterCount int               `json:"supporter_count"`
	DurationType   string            `json:"duration_type"`
	StartDate      string            `json:"start_date"`
	EndDate        *string           `json:"end_date"`
	Status         string            `json:"status"`
	ImageURL       *string           `json:"image_url"`
	Plans          []projectPlanJSON `json:"plans"`
	Seller         *userJSON         `json:"seller,omitempty"`
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
	var (
		items []projectSummaryJSON
		i     int
	)
	items = make([]projectSummaryJSON, 0, len(rows))
	for i = 0; i < len(rows); i++ {
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
	var (
		out    projectSummaryJSON
		seller userJSON
	)
	if v == nil {
		return projectSummaryJSON{}
	}
	out = projectItemFieldsJSON(v.Project)
	out.Plans = projectPlansToJSON(v.Plans)
	if v.Seller != nil {
		seller = userToJSON(*v.Seller)
		out.Seller = &seller
	}
	return out
}

func projectItemToJSON(p usecase.ProjectItemView) projectSummaryJSON {
	return projectItemFieldsJSON(p)
}

func projectItemFieldsJSON(p usecase.ProjectItemView) projectSummaryJSON {
	var (
		out projectSummaryJSON
		s   string
	)
	out = projectSummaryJSON{
		ID:             p.ID.String(),
		Title:          p.Title,
		Description:    p.Description,
		TargetAmount:   p.TargetAmount,
		CurrentAmount:  p.CurrentAmount,
		SupporterCount: p.SupporterCount,
		DurationType:   p.DurationType,
		StartDate:      p.StartDate.UTC().Format(time.RFC3339),
		Status:         p.Status,
		ImageURL:       p.MainImageURL,
	}
	if p.EndDate != nil {
		s = p.EndDate.UTC().Format(time.RFC3339)
		out.EndDate = &s
	}
	return out
}

func projectPlansToJSON(plans []usecase.ProjectPlanView) []projectPlanJSON {
	var (
		out []projectPlanJSON
		i   int
	)
	out = make([]projectPlanJSON, 0, len(plans))
	for i = 0; i < len(plans); i++ {
		out = append(out, projectPlanViewToJSON(plans[i]))
	}
	return out
}

func projectPlanViewToJSON(plan usecase.ProjectPlanView) projectPlanJSON {
	return projectPlanJSON{
		ID:             plan.ID.String(),
		ProjectID:      plan.ProjectID.String(),
		MembershipType: plan.MembershipType,
		Price:          plan.Price,
		MaxSupporters:  plan.MaxSupporters,
		Benefits:       plan.Benefits,
		IsActive:       plan.IsActive,
		SupporterCount: plan.SupporterCount,
	}
}

func projectImagesToJSON(images []usecase.ProjectImageView) []projectImageJSON {
	var (
		out []projectImageJSON
		i   int
	)
	out = make([]projectImageJSON, 0, len(images))
	for i = 0; i < len(images); i++ {
		out = append(out, projectImageJSON{
			ID:       images[i].ID.String(),
			ImageURL: images[i].ImageURL,
			IsMain:   images[i].IsMain,
			Order:    images[i].SortOrder,
		})
	}
	return out
}
