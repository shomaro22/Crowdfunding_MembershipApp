package controller

import (
	"fmt"
	"time"

	"crowdfund-app/usecase"
)

type createProjectJSON struct {
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	TargetAmount int64                `json:"target_amount"`
	DurationType string               `json:"duration_type"`
	StartDate    string               `json:"start_date"`
	EndDate      *string              `json:"end_date"`
	ImageURL     *string              `json:"image_url"`
	Images       []projectImageInJSON `json:"images"`
	Plans        []planInJSON         `json:"plans"`
}

type updateProjectJSON struct {
	Title        *string               `json:"title"`
	Description  *string               `json:"description"`
	TargetAmount *int64                `json:"target_amount"`
	DurationType *string               `json:"duration_type"`
	StartDate    *string               `json:"start_date"`
	EndDate      *string               `json:"end_date"`
	ImageURL     *string               `json:"image_url"`
	Images       *[]projectImageInJSON `json:"images"`
	IsReapply    bool                  `json:"is_reapply"`
}

type projectImageInJSON struct {
	ImageURL string `json:"image_url"`
	IsMain   bool   `json:"is_main"`
	Order    int    `json:"order"`
}

type planInJSON struct {
	MembershipType string `json:"membership_type"`
	Price          int    `json:"price"`
	MaxSupporters  *int   `json:"max_supporters"`
	Benefits       string `json:"benefits"`
}

func (b createProjectJSON) toInput() (usecase.CreateProjectIn, error) {
	var (
		start time.Time
		err   error
		end   *time.Time
		t     time.Time
		plans []usecase.PlanIn
		i     int
		p     planInJSON
	)
	start, err = time.Parse(time.RFC3339, b.StartDate)
	if err != nil {
		return usecase.CreateProjectIn{}, fmt.Errorf("invalid start_date: %w", err)
	}
	if b.EndDate != nil && *b.EndDate != "" {
		t, err = time.Parse(time.RFC3339, *b.EndDate)
		if err != nil {
			return usecase.CreateProjectIn{}, fmt.Errorf("invalid end_date: %w", err)
		}
		end = &t
	}
	plans = make([]usecase.PlanIn, 0, len(b.Plans))
	for i = range b.Plans {
		p = b.Plans[i]
		plans = append(plans, usecase.PlanIn{
			MembershipType: p.MembershipType,
			Price:          p.Price,
			MaxSupporters:  p.MaxSupporters,
			Benefits:       p.Benefits,
		})
	}
	return usecase.CreateProjectIn{
		Title:        b.Title,
		Description:  b.Description,
		TargetAmount: b.TargetAmount,
		DurationType: b.DurationType,
		StartDate:    start,
		EndDate:      end,
		ImageURL:     b.ImageURL,
		Images:       projectImagesInToUsecase(b.Images),
		Plans:        plans,
	}, nil
}

func (b updateProjectJSON) toInput() (usecase.UpdateProjectIn, error) {
	var (
		in     usecase.UpdateProjectIn
		tStart time.Time
		tEnd   time.Time
		err    error
	)
	in = usecase.UpdateProjectIn{
		Title:        b.Title,
		Description:  b.Description,
		TargetAmount: b.TargetAmount,
		ImageURL:     b.ImageURL,
		IsReapply:    b.IsReapply,
	}
	if b.Images != nil {
		imgs := projectImagesInToUsecase(*b.Images)
		in.Images = &imgs
	}
	if b.DurationType != nil {
		in.DurationType = b.DurationType
	}
	if b.StartDate != nil {
		tStart, err = time.Parse(time.RFC3339, *b.StartDate)
		if err != nil {
			return usecase.UpdateProjectIn{}, fmt.Errorf("invalid start_date: %w", err)
		}
		in.StartDate = &tStart
	}
	if b.EndDate != nil {
		if *b.EndDate == "" {
			in.EndDate = nil
		} else {
			tEnd, err = time.Parse(time.RFC3339, *b.EndDate)
			if err != nil {
				return usecase.UpdateProjectIn{}, fmt.Errorf("invalid end_date: %w", err)
			}
			in.EndDate = &tEnd
		}
	}
	return in, nil
}

func projectImagesInToUsecase(rows []projectImageInJSON) []usecase.ImageIn {
	var (
		out []usecase.ImageIn
		i   int
	)
	out = make([]usecase.ImageIn, 0, len(rows))
	for i = range rows {
		out = append(out, usecase.ImageIn{
			ImageURL: rows[i].ImageURL,
			IsMain:   rows[i].IsMain,
			Order:    rows[i].Order,
		})
	}
	return out
}
