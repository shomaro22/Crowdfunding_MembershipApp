package validator

import (
	"github.com/google/uuid"
)

type SellerSupporterListInput struct {
	ProjectID *uuid.UUID
	Limit     int
	Offset    int
}

type SellerSupporterListNormalized struct {
	ProjectID *uuid.UUID
	Limit     int
	Offset    int
}

func ValidateSellerSupporterList(in SellerSupporterListInput) (SellerSupporterListNormalized, error) {
	var (
		out SellerSupporterListNormalized
		err error
	)
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return SellerSupporterListNormalized{}, err
	}
	out.ProjectID = in.ProjectID
	return out, nil
}
