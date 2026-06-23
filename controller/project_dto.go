package controller

import (
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/usecase"
)

type createProjectJSON struct {
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	TargetAmount int64        `json:"target_amount"`
	DurationType string       `json:"duration_type"`
	StartDate    string       `json:"start_date"`
	EndDate      *string      `json:"end_date"`
	ImageURL     *string      `json:"image_url"`
	Plans        []planInJSON `json:"plans"`
}

type updateProjectJSON struct {
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	TargetAmount *int64  `json:"target_amount"`
	DurationType *string `json:"duration_type"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	ImageURL     *string `json:"image_url"`
	IsReapply    bool    `json:"is_reapply"`
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
		return usecase.CreateProjectIn{}, err
	}
	if b.EndDate != nil && *b.EndDate != "" {
		t, err = time.Parse(time.RFC3339, *b.EndDate)
		if err != nil {
			return usecase.CreateProjectIn{}, err
		}
		end = &t
	}
	plans = make([]usecase.PlanIn, 0, len(b.Plans))
	for i = range b.Plans {
		p = b.Plans[i]
		plans = append(plans, usecase.PlanIn{
			MembershipType: entity.MembershipType(p.MembershipType),
			Price:          p.Price,
			MaxSupporters:  p.MaxSupporters,
			Benefits:       p.Benefits,
		})
	}
	return usecase.CreateProjectIn{
		Title:        b.Title,
		Description:  b.Description,
		TargetAmount: b.TargetAmount,
		DurationType: entity.DurationType(b.DurationType),
		StartDate:    start,
		EndDate:      end,
		ImageURL:     b.ImageURL,
		Plans:        plans,
	}, nil
}

func (b updateProjectJSON) toInput() usecase.UpdateProjectIn {
	var (
		in     usecase.UpdateProjectIn
		d      entity.DurationType
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
	if b.DurationType != nil {
		d = entity.DurationType(*b.DurationType)
		in.DurationType = &d
	}
	if b.StartDate != nil {
		tStart, err = time.Parse(time.RFC3339, *b.StartDate)
		if err == nil {
			in.StartDate = &tStart
		}
	}
	if b.EndDate != nil {
		if *b.EndDate == "" {
			in.EndDate = nil
		} else {
			tEnd, err = time.Parse(time.RFC3339, *b.EndDate)
			if err == nil {
				in.EndDate = &tEnd
			}
		}
	}
	return in
}
