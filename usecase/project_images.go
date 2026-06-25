package usecase

import (
	"strings"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func mapEntityErr(err error) error {
	switch err {
	case entity.ErrBadTitle, entity.ErrBadProject, entity.ErrBadDuration, entity.ErrBadMembership, entity.ErrBadPrice, entity.ErrBadImages:
		return ErrBadRequest
	default:
		return err
	}
}

func buildProjectImages(ins []ImageIn, legacyURL *string, now time.Time) ([]entity.ProjectImage, *string, error) {
	var (
		inputs  []entity.ImageInput
		images  []entity.ProjectImage
		mainURL *string
		err     error
		i       int
		url     string
		orders  []int
	)

	if len(ins) > 0 {
		inputs = make([]entity.ImageInput, len(ins))
		for i = range ins {
			inputs[i] = entity.ImageInput{
				ImageURL: ins[i].ImageURL,
				IsMain:   ins[i].IsMain,
				Order:    ins[i].Order,
			}
		}
	} else if legacyURL != nil && strings.TrimSpace(*legacyURL) != "" {
		url = strings.TrimSpace(*legacyURL)
		inputs = []entity.ImageInput{{ImageURL: url, IsMain: true, Order: 0}}
	}
	if err = entity.ValidateProjectImages(inputs); err != nil {
		return nil, nil, err
	}
	if len(inputs) == 0 {
		return nil, nil, nil
	}
	orders = imageSortOrders(inputs)
	images = make([]entity.ProjectImage, len(inputs))
	for i = range inputs {
		url = strings.TrimSpace(inputs[i].ImageURL)
		images[i] = entity.ProjectImage{
			ID:        uuid.New(),
			ImageURL:  url,
			IsMain:    inputs[i].IsMain,
			SortOrder: orders[i],
			CreatedAt: now,
		}
		if inputs[i].IsMain {
			mainURL = &url
		}
	}
	return images, mainURL, nil
}

func imageSortOrders(inputs []entity.ImageInput) []int {
	var (
		orders  = make([]int, len(inputs))
		allZero = true
		i       int
	)
	for i = range inputs {
		orders[i] = inputs[i].Order
		if inputs[i].Order != 0 {
			allZero = false
		}
	}
	if allZero {
		for i = range orders {
			orders[i] = i
		}
	}
	return orders
}
