package base

import "github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"

var transitions = map[types.BoletoStatus]map[types.BoletoStatus]bool{
	types.StatusCreated: {
		types.StatusProcessing: true,
		types.StatusCancelled:  true,
	},
	types.StatusProcessing: {
		types.StatusIssued:    true,
		types.StatusFailed:    true,
		types.StatusCancelled: true,
	},
	types.StatusIssued: {
		types.StatusPaid:      true,
		types.StatusPartial:   true,
		types.StatusExpired:   true,
		types.StatusCancelled: true,
	},
	types.StatusPartial: {
		types.StatusPaid: true,
	},
}

func IsKnownStatus(status types.BoletoStatus) bool {
	switch status {
	case types.StatusCreated, types.StatusProcessing, types.StatusIssued, types.StatusFailed,
		types.StatusCancelled, types.StatusPartial, types.StatusPaid, types.StatusExpired:
		return true
	default:
		return false
	}
}

func CanTransition(from, to types.BoletoStatus) bool {
	if from == to {
		return true
	}
	return transitions[from][to]
}
