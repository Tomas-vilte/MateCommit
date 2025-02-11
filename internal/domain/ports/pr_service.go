package ports

import "context"

// PRService define la interfaz para el servicio de resumen de Pull Requests.
type PRService interface {
	SummarizePR(ctx context.Context, prNumber int) (string, error)
}
