package payment

import "context"

type Result string

const (
	ResultSuccess Result = "success"
	ResultFailed  Result = "failed"
	ResultTimeout Result = "timeout"
)

type Service interface {
	Pay(
		ctx context.Context,
		orderID int64,
		amount int64,
		currency string,
	) (Result, error)
}

type FakeService struct {
	Result Result
}

func NewFakeService(result Result) *FakeService {
	return &FakeService{
		Result: result,
	}
}

func (s *FakeService) Pay(
	ctx context.Context,
	orderID int64,
	amount int64,
	currency string,
) (Result, error) {
	return s.Result, nil
}
